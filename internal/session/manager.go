package session

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"party-buddy/internal/db"
	"party-buddy/internal/util"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	db      *db.DBPool
	storage SyncStorage
	runChan chan runMsg
}

func NewManager(db *db.DBPool) *Manager {
	return &Manager{
		db:      db,
		storage: NewSyncStorage(),
		runChan: make(chan runMsg),
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
					log.Default().Writer(),
					fmt.Sprintf("sessionUpdater(sid %s)", msg.sid),
					log.Default().Flags(),
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

func (m *Manager) registerImage(ctx context.Context, tx pgx.Tx, sid SessionID, imageID ImageID) error {
	if imageID.Valid {
		if err := db.CreateSessionImageRef(ctx, tx, sid.UUID(), imageID.UUID); err != nil {
			return fmt.Errorf("could not register an image (id %s) for session %s: %w", imageID, sid, err)
		}
	}

	return nil
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
		player = util.Must(s.addPlayer(sid, clientID, nickname, tx))
	})

	if err != nil {
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

	db.RemoveSessionImageRefs(ctx, tx, sid.UUID())
	s.closeUpdater(sid)
	s.removeSession(sid)
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(tx TxChan, message ServerTx) {
	if tx != nil {
		tx <- message
	}
}

func (m *Manager) closePlayerTx(s *UnsafeStorage, sid SessionID, playerID PlayerID) bool {
	return s.closePlayerTx(sid, playerID)
}

func (m *Manager) makeMsgError(ctx context.Context, err error) ServerTx {
	return &MsgError{
		baseTx: baseTx{Ctx: ctx},
		Inner:  err,
	}
}

func (m *Manager) makeMsgJoined(
	ctx context.Context,
	playerID PlayerID,
	sid SessionID,
	game *Game,
) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStatus(ctx context.Context, players []Player) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskStart(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	task Task,
	answer TaskAnswer,
) ServerTx {
	msg := &MsgTaskStart{
		baseTx:   baseTx{Ctx: ctx},
		TaskIdx:  taskIdx,
		Deadline: deadline,
	}
	switch t := task.(type) {
	case ChoiceTask:
		msg.Options = &t.Options
		return msg
	case PhotoTask:
		i := ImageID(answer.(PhotoTaskAnswer))
		msg.ImgID = &i
		return msg
	default:
		return msg
	}

}

func (m *Manager) makeMsgPollStart(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	options []PollOption,
) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskEnd(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	results []AnswerResult,
) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStart(ctx context.Context, deadline time.Time) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgWaiting(ctx context.Context, playersReady map[PlayerID]struct{}) ServerTx {
	// TODO
	return nil
}

func (m *Manager) sendMsgErrorToAllPlayers(ctx context.Context, sid SessionID, s *UnsafeStorage, err error) {
	for _, tx := range s.PlayerTxs(sid) {
		m.sendToPlayer(tx, m.makeMsgError(ctx, err))
	}
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
		_, err = s.PlayerByID(sid, playerID)
		if err != nil {
			err = ErrNoPlayer
			return
		}
		if answer != nil {
			// during validation, we checked that provided value of answer matched provided answer type
			// now we are checking that task type matches provided answer type
			task := s.getTaskByIdx(sid, taskIdx)
			if task == nil {
				err = ErrNoTask
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
