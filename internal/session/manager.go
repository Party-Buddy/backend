package session

import (
	"context"
	"fmt"
	"log"
	"party-buddy/internal/db"
	"party-buddy/internal/util"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	db       *db.DBPool
	storage  SyncStorage
	updaters map[SessionID]chan<- updateMsg
	runChan  chan runMsg
}

func NewManager(db *db.DBPool) *Manager {
	return &Manager{
		db:       db,
		storage:  NewSyncStorage(),
		runChan:  make(chan runMsg),
		updaters: make(map[SessionID]chan<- updateMsg),
	}
}

func (m *Manager) Storage() *SyncStorage {
	return &m.storage
}

// # Update logic
//
// A manager runs a number of goroutines — one for each session, to be precise.
// Communication happens through the channel stored in m.updaters.
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
					m:   m,
					sid: msg.sid,
					rx:  msg.rx,
					log: logger,
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
// Consider using lockStorageThenUpdate instead if you need access to the storage.
func (m *Manager) sendToUpdater(sid SessionID, msg updateMsg) {
	var tx chan<- updateMsg
	var ok bool
	m.storage.Atomically(func(s *UnsafeStorage) {
		// FIXME: move updaters to storage
		tx, ok = m.updaters[sid]
	})

	if !ok {
		return
	}

	if msg == nil {
		m.storage.Atomically(func(s *UnsafeStorage) {
			close(tx)
			delete(m.updaters, sid)
		})
	} else {
		tx <- msg
	}
}

// lockStorageThenUpdate locks the storage and runs the function.
// Once it returns, the storage is unlocked and the messages are sent to the updater.
func (m *Manager) lockStorageThenUpdate(sid SessionID, f func(s *UnsafeStorage) []updateMsg) {
	var msgs []updateMsg
	m.storage.Atomically(func(s *UnsafeStorage) {
		msgs = f(s)
	})

	for _, msg := range msgs {
		m.sendToUpdater(sid, msg)
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
		sid, code, err = s.newSession(game, owner, requireReady, playersMax, deadline)
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

		updateChan = make(chan updateMsg)
		m.updaters[sid] = updateChan
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
) (playerID PlayerID, err error) {
	m.lockStorageThenUpdate(sid, func(s *UnsafeStorage) []updateMsg {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return nil
		}
		if player, err := s.PlayerByClientID(sid, clientID); err == nil {
			playerID = player.ID
			return m.reconnect(ctx, s, sid, &player, tx)
		}
		if !s.AwaitingPlayers(sid) {
			err = ErrGameInProgress
			return nil
		}
		if s.ClientBanned(sid, clientID) {
			err = ErrClientBanned
			return nil
		}
		if s.HasPlayerNickname(sid, nickname) {
			err = ErrNicknameUsed
			return nil
		}
		if s.SessionFull(sid) {
			err = ErrLobbyFull
			return nil
		}

		player := util.Must(s.addPlayer(sid, clientID, nickname, tx))
		playerID = player.ID
		return m.join(ctx, s, sid, &player, false)
	})

	return
}

// RemovePlayer removes a player from a session.
func (m *Manager) RemovePlayer(ctx context.Context, sid SessionID, playerID PlayerID) {
	m.sendToUpdater(sid, &updateMsgRemovePlayer{
		ctx:      ctx,
		playerID: playerID,
	})
}

// # Event handlers

func (m *Manager) reconnect(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionID,
	player *Player,
	tx TxChan,
) []updateMsg {
	m.sendToPlayer(player.Tx, m.makeMsgError(ctx, ErrReconnected))
	m.closePlayerTx(s, sid, player.ID)

	return m.join(ctx, s, sid, player, true)
}

func (m *Manager) join(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionID,
	player *Player,
	reconnect bool,
) []updateMsg {
	game, _ := s.SessionGame(sid)
	joined := m.makeMsgJoined(ctx, player.ID, sid, &game)
	m.sendToPlayer(player.Tx, joined)

	gameStatus := m.makeMsgGameStatus(ctx, s.Players(sid))

	if reconnect {
		m.sendToPlayer(player.Tx, gameStatus)
	} else {
		for _, tx := range s.PlayerTxs(sid) {
			m.sendToPlayer(tx, gameStatus)
		}
	}

	// TODO: move to updater
	var stateMessage ServerTx
	switch state := s.sessionState(sid).(type) {
	case *AwaitingPlayersState:
		stateMessage = m.makeMsgWaiting(ctx, state.playersReady)
	case *GameStartedState:
		stateMessage = m.makeMsgGameStart(ctx, state.deadline)
	case *TaskStartedState:
		stateMessage = m.makeMsgTaskStart(ctx, state.taskIdx, state.deadline)
	case *PollStartedState:
		stateMessage = m.makeMsgPollStart(ctx, state.taskIdx, state.deadline, state.options)
	case *TaskEndedState:
		stateMessage = m.makeMsgTaskEnd(ctx, state.taskIdx, state.deadline, state.results)
	}
	m.sendToPlayer(player.Tx, stateMessage)

	return []updateMsg{
		&updateMsgPlayerAdded{
			ctx:      ctx,
			playerID: player.ID,
		},
	}
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

	if updater := m.updaters[sid]; updater != nil {
		close(updater)
	}
	delete(m.updaters, sid)

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

func (m *Manager) makeMsgTaskStart(ctx context.Context, taskIdx int, deadline time.Time) ServerTx {
	// TODO
	return nil
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
