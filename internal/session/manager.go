package session

import (
	"context"
	"fmt"
	"party-buddy/internal/db"
	"party-buddy/internal/session/storage"
	"party-buddy/internal/util"
	"time"

	"github.com/jackc/pgx/v5"
)

type Manager struct {
	db      *db.DBPool
	storage storage.SyncStorage
}

func NewManager(db *db.DBPool) *Manager {
	return &Manager{
		db:      db,
		storage: storage.SyncStorage{},
	}
}

// NewSession creates a new session.
//
// Assumes all values are valid.
func (m *Manager) NewSession(
	ctx context.Context,
	game *storage.Game,
	owner storage.ClientId,
	ownerNickname string,
	requireReady bool,
	playersMax int,
) (sid storage.SessionId, code storage.InviteCode, ownerId storage.PlayerId, err error) {
	m.storage.Atomically(func(s *storage.UnsafeStorage) {
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

	return
}

func (m *Manager) registerImage(ctx context.Context, tx pgx.Tx, sid storage.SessionId, imageId storage.ImageId) error {
	if imageId.Valid {
		if err := db.CreateSessionImageRef(ctx, tx, sid.UUID(), imageId.UUID); err != nil {
			return fmt.Errorf("could not register an image (id %s) for session %s: %w", imageId, sid, err)
		}
	}

	return nil
}

func (m *Manager) JoinSession(
	ctx context.Context,
	sid storage.SessionId,
	clientId storage.ClientId,
	nickname string,
	tx storage.TxChan,
) (playerId storage.PlayerId, err error) {
	m.storage.Atomically(func(s *storage.UnsafeStorage) {
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
	s *storage.UnsafeStorage,
	sid storage.SessionId,
	player *storage.Player,
	tx storage.TxChan,
) {
	// TODO: handle reconnects

	// TODO: tell the client why we're disconnecting them
	m.closePlayerTx(ctx, s, sid, player.Id)

	m.onJoin(ctx, s, sid, player, true)
}

func (m *Manager) onJoin(
	ctx context.Context,
	s *storage.UnsafeStorage,
	sid storage.SessionId,
	player *storage.Player,
	reconnect bool,
) {
	game, _ := s.SessionGame(sid)
	joined := m.makeMsgJoined(player.Id, sid, &game)
	m.sendToPlayer(ctx, player.Tx, joined)

	players := s.Players(sid)
	gameStatus := m.makeMsgGameStatus(players)

	if reconnect {
		m.sendToPlayer(ctx, player.Tx, gameStatus)
	} else {
		for _, tx := range s.PlayerTxs(sid) {
			m.sendToPlayer(ctx, tx, gameStatus)
		}
	}

	var stateMessage any
	switch state := s.SessionState(sid).(type) {
	case *storage.AwaitingPlayersState:
		stateMessage = m.makeMsgWaiting(state.PlayersReady)
	case *storage.GameStartedState:
		stateMessage = m.makeMsgGameStart(state.Deadline)
	case *storage.TaskStartedState:
		stateMessage = m.makeMsgTaskStart(state.TaskIdx, state.Deadline)
	case *storage.PollStartedState:
		stateMessage = m.makeMsgPollStart(state.TaskIdx, state.Deadline, state.Options)
	case *storage.TaskEndedState:
		stateMessage = m.makeMsgTaskEnd(state.TaskIdx, state.Deadline, state.Results)
	}
	m.sendToPlayer(ctx, player.Tx, stateMessage)

	// TODO: notify the websockets handler of the current state
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(ctx context.Context, tx storage.TxChan, message any) {
	// TODO: type message appropriately
	// TODO: send a message to the client's websocket handler somehow
}

func (m *Manager) closePlayerTx(
	ctx context.Context,
	s *storage.UnsafeStorage,
	sid storage.SessionId,
	playerId storage.PlayerId,
) {
	// TODO
}

func (m *Manager) makeMsgJoined(
	playerId storage.PlayerId,
	sid storage.SessionId,
	game *storage.Game,
) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStatus(players []storage.Player) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskStart(taskIdx int, deadline time.Time) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgPollStart(taskIdx int, deadline time.Time, options []storage.PollOption) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskEnd(taskIdx int, deadline time.Time, results []storage.AnswerResult) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStart(deadline time.Time) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgWaiting(playersReady map[storage.PlayerId]struct{}) any {
	// TODO
	return nil
}
