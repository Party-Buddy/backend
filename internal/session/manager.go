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

const rxQueueCapacity int = 10

type Manager struct {
	db      *db.DBPool
	storage SyncStorage
	runChan chan runMsg
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

	m.runChan <- &runMsgSpawn{
		sid: sid,
		// FIXME: stuff that channel into somewhere so we can tell the updater to update
		rx: nil,
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
	case *AwaitingPlayersState:
		stateMessage = m.makeMsgWaiting(state.PlayersReady)
	case *GameStartedState:
		stateMessage = m.makeMsgGameStart(state.Deadline)
	case *TaskStartedState:
		stateMessage = m.makeMsgTaskStart(state.TaskIdx, state.Deadline)
	case *PollStartedState:
		stateMessage = m.makeMsgPollStart(state.TaskIdx, state.Deadline, state.Options)
	case *TaskEndedState:
		stateMessage = m.makeMsgTaskEnd(state.TaskIdx, state.Deadline, state.Results)
	}
	m.sendToPlayer(ctx, player.Tx, stateMessage)

	// TODO: notify the websockets handler of the current state
	// TODO: notify the sessionUpdater of a new arrival
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(ctx context.Context, tx TxChan, message any) {
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
	playerId PlayerId,
	sid SessionId,
	game *Game,
) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStatus(players []Player) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskStart(taskIdx int, deadline time.Time) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgPollStart(taskIdx int, deadline time.Time, options []PollOption) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskEnd(taskIdx int, deadline time.Time, results []AnswerResult) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgGameStart(deadline time.Time) any {
	// TODO
	return nil
}

func (m *Manager) makeMsgWaiting(playersReady map[PlayerId]struct{}) any {
	// TODO
	return nil
}
