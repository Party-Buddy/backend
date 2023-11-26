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

func (m *Manager) sendToUpdater(sid SessionID, msg updateMsg) {
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

// RemovePlayer removes a player from a session.
func (m *Manager) RemovePlayer(ctx context.Context, sid SessionID, clientID ClientID, playerID PlayerID) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		m.closePlayerTx(ctx, s, sid, playerID)

		_, ok := s.removePlayer(sid, clientID)
		if ok {
			// TODO: notify other players that the player disconnected
		}
	})
}

// NewSession creates a new session.
//
// Assumes all values are valid.
func (m *Manager) NewSession(
	ctx context.Context,
	tx pgx.Tx,
	game *Game,
	owner ClientID,
	ownerNickname string,
	requireReady bool,
	playersMax int,
) (sid SessionID, code InviteCode, ownerID PlayerID, err error) {
	m.storage.Atomically(func(s *UnsafeStorage) {
		sid, code, ownerID, err = s.NewSession(game, owner, ownerNickname, requireReady, playersMax)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				s.RemoveSession(sid)
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

	updateChan := make(chan updateMsg)
	m.updaters[sid] = updateChan
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
	m.storage.Atomically(func(s *UnsafeStorage) {
		if !s.SessionExists(sid) {
			err = ErrNoSession
			return
		}
		if player, err := s.PlayerByClientID(sid, clientID); err == nil {
			playerID = player.ID
			m.onReconnect(ctx, s, sid, &player, tx)
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

		player := util.Must(s.AddPlayer(sid, clientID, nickname, tx))
		playerID = player.ID
		m.onJoin(ctx, s, sid, &player, false)
	})

	return
}

// # Event handlers

func (m *Manager) onReconnect(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionID,
	player *Player,
	tx TxChan,
) {
	// TODO: handle reconnects

	// TODO: tell the client why we're disconnecting them
	m.closePlayerTx(ctx, s, sid, player.ID)

	m.onJoin(ctx, s, sid, player, true)
}

func (m *Manager) onJoin(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionID,
	player *Player,
	reconnect bool,
) {
	game, _ := s.SessionGame(sid)
	joined := m.makeMsgJoined(ctx, player.ID, sid, &game)
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

	m.sendToUpdater(sid, &updateMsgPlayerAdded{playerID: player.ID})
}

// # Server-to-client communication

func (m *Manager) sendToPlayer(tx TxChan, message ServerTx) {
	// TODO: type message appropriately
	// TODO: send a message to the client's websocket handler somehow
}

func (m *Manager) closePlayerTx(
	ctx context.Context,
	s *UnsafeStorage,
	sid SessionID,
	playerID PlayerID,
) {
	// TODO
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
