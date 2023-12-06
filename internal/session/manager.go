package session

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"party-buddy/internal/db"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	db      *db.DBPool
	storage SyncStorage
	runChan chan runMsg
	log     *log.Logger
}

func NewManager(db *db.DBPool, logger *log.Logger) *Manager {
	return &Manager{
		db:      db,
		storage: NewSyncStorage(),
		runChan: make(chan runMsg),
		log:     logger,
	}
}

func (m *Manager) Storage() *SyncStorage {
	return &m.storage
}

// # Update logic
//
// A manager runs a number of goroutines — one for each session, to be precise.
// Communication happens through the channel accessed via (*UnsafeStorage).updater().
//
// The rules are:
//   - You must not send a message to an updater while holding the storage locked.
//     This will lead to deadlocks.
//
//   - The communication must be strictly one-way.
//     The updater must not call into the manager's methods.
//     The only exception are utilities and server-to-client communication.
//
//   - If you need the current session state, that code belongs to the updater.

type runMsg interface {
	isRunMsg()
}

type runMsgSpawn struct {
	sid SessionID
	rx  <-chan updateMsg
}

func (*runMsgSpawn) isRunMsg() {}

func (m *Manager) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer

		case msg := <-m.runChan:
			switch msg := msg.(type) {
			case *runMsgSpawn:
				logger := log.New(
					m.log.Writer(),
					fmt.Sprintf("sessionUpdater(sid %s): ", msg.sid),
					m.log.Flags(),
				)
				updater := sessionUpdater{
					m:        m,
					sid:      msg.sid,
					rx:       msg.rx,
					log:      logger,
					deadline: time.NewTimer(NoOwnerTimeout),
				}
				group.Go(func() error {
					return updater.run(ctx)
				})
			}
		}
	}

	return group.Wait()
}

// sendToUpdater sends a message to a session updater goroutine.
//
// DANGER: you MUST NOT call this method while holding the storage's mutex,
// or you WILL get deadlocks.
func (m *Manager) sendToUpdater(sid SessionID, msg updateMsg) {
	if msg == nil {
		m.storage.Atomically(func(s *UnsafeStorage) {
			s.closeUpdater(sid)
		})
	} else {
		var updaterChan chan<- updateMsg
		m.storage.Atomically(func(s *UnsafeStorage) {
			updaterChan = s.updater(sid)
		})
		updaterChan <- msg
	}
}

// # DB access

func (m *Manager) registerImage(ctx context.Context, tx pgx.Tx, sid SessionID, imageID ImageID) error {
	if imageID.Valid {
		if err := db.CreateSessionImageRef(ctx, tx, sid.UUID(), imageID.UUID); err != nil {
			return fmt.Errorf("could not register an image (id %s) for session %s: %w", imageID, sid, err)
		}
	}

	return nil
}

func (m *Manager) newImgMetadataForSession(ctx context.Context, tx pgx.Tx, sid SessionID, clientID ClientID) (ImageID, error) {
	var err error
	var dbImgID uuid.NullUUID
	dbImgID, err = db.CreateImageMetadata(tx, ctx, clientID.UUID())
	if err != nil {
		return ImageID{}, err
	}
	err = db.CreateSessionImageRef(ctx, tx, sid.UUID(), dbImgID.UUID)
	if err != nil {
		return ImageID{}, err
	}
	return ImageID(dbImgID), nil
}

// # Synchronous methods

// NewSession creates a new session.
//
// Assumes all values are valid.
func (m *Manager) NewSession(
	ctx context.Context,
	tx pgx.Tx,
	game *Game,
	owner ClientID,
	requireReady bool,
	playersMax int,
) (sid SessionID, code InviteCode, err error) {
	var updateChan chan updateMsg

	m.storage.Atomically(func(s *UnsafeStorage) {
		deadline := time.Now().Add(NoOwnerTimeout)
		sid, code, updateChan, err = s.newSession(
			game, owner, requireReady, playersMax, deadline,
		)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				m.closeSession(ctx, s, tx, sid)
			}
		}()

		if err = m.registerImage(ctx, tx, sid, game.ImageID); err != nil {
			return
		}

		for _, task := range game.Tasks {
			if err = m.registerImage(ctx, tx, sid, task.GetImageID()); err != nil {
				return
			}
		}
	})

	if err != nil {
		return
	}

	m.runChan <- &runMsgSpawn{
		sid: sid,
		rx:  updateChan,
	}

	return
}

func (m *Manager) JoinSession(
	ctx context.Context,
	sid SessionID,
	clientID ClientID,
	nickname string,
	tx TxChan,
) (player Player, err error) {
	var reconnected bool

	m.storage.Atomically(func(s *UnsafeStorage) {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return
		}
		if player, err = s.PlayerByClientID(sid, clientID); err == nil {
			reconnected = true
			m.sendToPlayer(player.tx, m.makeMsgError(ctx, ErrReconnected))
			m.closePlayerTx(s, sid, player.ID)
			return
		}
		if !s.AwaitingPlayers(sid) {
			err = ErrGameInProgress
			return
		}
		if s.ClientBanned(sid, clientID) {
			err = ErrClientBanned
			return
		}
		if s.HasPlayerNickname(sid, nickname) {
			err = ErrNicknameUsed
			return
		}
		if s.SessionFull(sid) {
			err = ErrLobbyFull
			return
		}

		// note: we must add the player inside the critical section.
		// we don't want to end up accepting two simultaneous requests to join.
		if player, err = s.addPlayer(sid, clientID, nickname, tx); err != nil {
			err = fmt.Errorf("%w: could not add player to the session: %w", ErrInternal, err)
			return
		}
	})

	if err == nil {
		m.sendToUpdater(sid, &updateMsgPlayerAdded{
			ctx:         ctx,
			playerID:    player.ID,
			reconnected: reconnected,
		})
	}

	return
}

// RemovePlayer removes a player from a session.
func (m *Manager) RemovePlayer(ctx context.Context, sid SessionID, playerID PlayerID) {
	m.sendToUpdater(sid, &updateMsgRemovePlayer{
		ctx:      ctx,
		playerID: playerID,
	})
}

// SetPlayerReady sets the readiness of a player for the game.
func (m *Manager) SetPlayerReady(ctx context.Context, sid SessionID, playerID PlayerID, ready bool) (err error) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return
		}
		if !s.PlayerExists(sid, playerID) {
			err = ErrNoPlayer
			return
		}
	})

	if err != nil {
		return
	}

	m.sendToUpdater(sid, &updateMsgSetPlayerReady{
		ctx:      ctx,
		playerID: playerID,
		ready:    ready,
	})

	return
}

func (m *Manager) UpdatePlayerAnswer(
	ctx context.Context,
	sid SessionID,
	playerID PlayerID,
	answer TaskAnswer,
	ready bool,
	taskIdx int,
) error {
	var err error
	m.storage.Atomically(func(s *UnsafeStorage) {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return
		}
		if !s.PlayerExists(sid, playerID) {
			err = ErrNoPlayer
			return
		}
		if answer != nil {
			// during validation, we checked that provided value of answer matched provided answer type
			// now we are checking that task type matches provided answer type
			task := s.taskByIdx(sid, taskIdx)
			if task == nil {
				err = ErrTaskIndexOutOfBounds
				return
			}
			ok := false
			switch task.(type) {
			case ChoiceTask:
				_, ok = answer.(ChoiceTaskAnswer)
			case CheckedTextTask:
				_, ok = answer.(CheckedTextAnswer)
			case TextTask:
				_, ok = answer.(TextTaskAnswer)
			}
			if !ok {
				err = ErrTypesTaskAndAnswerMismatch
				return
			}
		}
	})
	if err != nil {
		return err
	}

	m.sendToUpdater(sid, &updateMsgUpdTaskAnswer{
		ctx:      ctx,
		playerID: playerID,
		answer:   answer,
		ready:    ready,
		taskIdx:  taskIdx,
	})
	return nil
}

// closeSession terminates the session and removes it from the storage.
//
// This method can be called by an updater.
func (m *Manager) closeSession(
	ctx context.Context,
	s *UnsafeStorage,
	tx pgx.Tx,
	sid SessionID,
) {
	s.ForEachPlayer(sid, func(p Player) {
		m.closePlayerTx(s, sid, p.ID)
	})

	if err := db.RemoveSessionImageRefs(ctx, tx, sid.UUID()); err != nil {
		m.log.Printf("while closing session %s: could not remove session image references: %s", sid, err)
	}

	s.closeUpdater(sid)
	s.removeSession(sid)
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(tx TxChan, message ServerTx) {
	if tx != nil {
		tx <- message
	}
}

func (m *Manager) sendToAllPlayers(s *UnsafeStorage, sid SessionID, message ServerTx) {
	for _, tx := range s.PlayerTxs(sid) {
		m.sendToPlayer(tx, message)
	}
}

func (m *Manager) sendErrorToAllPlayers(ctx context.Context, s *UnsafeStorage, sid SessionID, err error) {
	for _, tx := range s.PlayerTxs(sid) {
		m.sendToPlayer(tx, m.makeMsgError(ctx, err))
	}
}

func (m *Manager) closePlayerTx(s *UnsafeStorage, sid SessionID, playerID PlayerID) bool {
	return s.closePlayerTx(sid, playerID)
}
