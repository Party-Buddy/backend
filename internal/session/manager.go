package session

import (
	"context"
	"fmt"
	"party-buddy/internal/db"
	"party-buddy/internal/util"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	db       *db.DBPool
	storage  SyncStorage
	updaters map[SessionId]chan<- updateMsg
	runChan  chan runMsg
}

func NewManager(db *db.DBPool) *Manager {
	return &Manager{
		db:      db,
		storage: SyncStorage{},
		runChan: make(chan runMsg),
	}
}

// # Update logic

type runMsg interface {
	isRunMsg()
}

type runMsgSpawn struct {
	sid SessionId
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
				updater := sessionUpdater{
					m:   m,
					sid: msg.sid,
					rx:  msg.rx,
				}
				group.Go(func() error {
					return updater.run(ctx)
				})
			}
		}
	}

	return group.Wait()
}

func (m *Manager) sendToUpdater(sid SessionId, msg updateMsg) {
	tx, ok := m.updaters[sid]
	if !ok {
		return
	}

	if msg == nil {
		close(tx)
		delete(m.updaters, sid)
	} else {
		tx <- msg
	}
}

// # Synchronous methods

// NewSession creates a new session.
//
// Assumes all values are valid.
func (m *Manager) NewSession(
	ctx context.Context,
	game *Game,
	owner ClientId,
	ownerNickname string,
	requireReady bool,
	playersMax int,
) (sid SessionId, code InviteCode, ownerId PlayerId, err error) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		sid, code, ownerId, err = s.NewSession(game, owner, ownerNickname, requireReady, playersMax)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				s.RemoveSession(sid)
			}
		}()

		err = m.db.AcquireTx(ctx, func(tx pgx.Tx) error {
			if err = m.registerImage(ctx, tx, sid, game.ImageId); err != nil {
				return err
			}

			for _, task := range game.Tasks {
				if err = m.registerImage(ctx, tx, sid, task.ImageId()); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return
		}
	})

	if err != nil {
		return
	}

	updateChan := make(chan updateMsg)
	m.updaters[sid] = updateChan
	m.runChan <- &runMsgSpawn{
		sid: sid,
		rx:  updateChan,
	}

	return
}

func (m *Manager) registerImage(ctx context.Context, tx pgx.Tx, sid SessionId, imageId ImageId) error {
	if imageId.Valid {
		if err := db.CreateSessionImageRef(ctx, tx, sid.UUID(), imageId.UUID); err != nil {
			return fmt.Errorf("could not register an image (id %s) for session %s: %w", imageId, sid, err)
		}
	}

	return nil
}

func (m *Manager) JoinSession(
	ctx context.Context,
	sid SessionId,
	clientId ClientId,
	nickname string,
	tx TxChan,
) (playerId PlayerId, err error) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return
		}
		if player, err := s.PlayerByClientId(sid, clientId); err == nil {
			playerId = player.Id
			m.onReconnect(ctx, s, sid, &player, tx)
			return
		}
		if !s.AwaitingPlayers(sid) {
			err = ErrGameInProgress
			return
		}
		if s.ClientBanned(sid, clientId) {
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

		player := util.Must(s.AddPlayer(sid, clientId, nickname, tx))
		playerId = player.Id
		m.onJoin(ctx, s, sid, &player, false)
	})

	return
}

// # Event handlers

func (m *Manager) onReconnect(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionId,
	player *Player,
	tx TxChan,
) {
	// TODO: handle reconnects

	// TODO: tell the client why we're disconnecting them
	m.closePlayerTx(ctx, s, sid, player.Id)

	m.onJoin(ctx, s, sid, player, true)
}

func (m *Manager) onJoin(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionId,
	player *Player,
	reconnect bool,
) {
	game, _ := s.SessionGame(sid)
	joined := m.makeMsgJoined(ctx, player.Id, sid, &game)
	m.sendToPlayer(player.Tx, joined)

	players := s.Players(sid)
	gameStatus := m.makeMsgGameStatus(ctx, players)

	if reconnect {
		m.sendToPlayer(player.Tx, gameStatus)
	} else {
		for _, tx := range s.PlayerTxs(sid) {
			m.sendToPlayer(tx, gameStatus)
		}
	}

	var stateMessage ServerTx
	switch state := s.SessionState(sid).(type) {
	case *AwaitingPlayersState:
		stateMessage = m.makeMsgWaiting(ctx, state.PlayersReady)
	case *GameStartedState:
		stateMessage = m.makeMsgGameStart(ctx, state.Deadline)
	case *TaskStartedState:
		stateMessage = m.makeMsgTaskStart(ctx, state.TaskIdx, state.Deadline)
	case *PollStartedState:
		stateMessage = m.makeMsgPollStart(ctx, state.TaskIdx, state.Deadline, state.Options)
	case *TaskEndedState:
		stateMessage = m.makeMsgTaskEnd(ctx, state.TaskIdx, state.Deadline, state.Results)
	}
	m.sendToPlayer(player.Tx, stateMessage)

	// TODO: notify the websockets handler of the current state

	m.sendToUpdater(sid, &updateMsgPlayerAdded{playerId: player.Id})
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(tx TxChan, message ServerTx) {
	// TODO: type message appropriately
	// TODO: send a message to the client's websocket handler somehow
}

func (m *Manager) closePlayerTx(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionId,
	playerId PlayerId,
) {
	// TODO
}

func (m *Manager) makeMsgJoined(
	ctx context.Context,
	playerId PlayerId,
	sid SessionId,
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

func (m *Manager) makeMsgWaiting(ctx context.Context, playersReady map[PlayerId]struct{}) ServerTx {
	// TODO
	return nil
}

func (m *Manager) SidByInviteCode(code string) (sid SessionId, ok bool) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		sid, ok = s.SidByInviteCode(InviteCode(code))
	})
	return
}

func (m *Manager) SessionExists(sid SessionId) (ok bool) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		ok = s.SessionExists(sid)
	})
	return
}

func (m *Manager) RequestDisconnect(ctx context.Context, sid SessionId, clientID ClientId, playerID PlayerId) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		m.closePlayerTx(ctx, s, sid, playerID)

		_, ok := s.removePlayer(sid, clientID)
		if ok {
			// TODO: notify other players that the player disconnected
		}
	})
}
