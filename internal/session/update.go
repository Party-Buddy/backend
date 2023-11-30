package session

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// # Update messages

type updateMsg interface {
	isUpdateMsg()
}

type updateMsgPlayerAdded struct {
	ctx         context.Context
	playerID    PlayerID
	reconnected bool
}

func (*updateMsgPlayerAdded) isUpdateMsg() {}

type updateMsgRemovePlayer struct {
	ctx      context.Context
	playerID PlayerID
}

func (*updateMsgRemovePlayer) isUpdateMsg() {}

type updateMsgChangeStateTo struct {
	nextState State
}

func (updateMsgChangeStateTo) isUpdateMsg() {}

// # Run logic

type sessionUpdater struct {
	m        *Manager
	sid      SessionID
	rx       <-chan updateMsg
	log      *log.Logger
	deadline *time.Timer
}

func (u *sessionUpdater) run(ctx context.Context) error {
	u.m.storage.Atomically(func(s *UnsafeStorage) {
		u.changeStateTo(ctx, s, s.sessionState(u.sid))
	})

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-u.deadline.C:
			u.m.storage.Atomically(func(s *UnsafeStorage) {
				u.deadlineExpired(ctx, s)
			})

		case msg := <-u.rx:
			if msg == nil {
				return nil
			}

			u.m.storage.Atomically(func(s *UnsafeStorage) {
				switch msg := msg.(type) {
				case *updateMsgPlayerAdded:
					u.playerAdded(msg.ctx, s, msg.playerID, msg.reconnected)
				case *updateMsgRemovePlayer:
					u.removePlayer(msg.ctx, s, msg.playerID)
				case *updateMsgChangeStateTo:
					u.changeStateTo(ctx, s, msg.nextState)
				}
			})
		}
	}
}

func (u *sessionUpdater) playerAdded(
	ctx context.Context,
	s *UnsafeStorage,
	playerID PlayerID,
	reconnected bool,
) {
	player, err := s.PlayerByID(u.sid, playerID)
	if err != nil {
		u.log.Printf("while handling added player: %s", err)
		return
	}

	state := s.sessionState(u.sid)
	if state == nil {
		return
	}

	if state, ok := state.(*AwaitingPlayersState); ok && state.owner == player.ClientID {
		// the owner has at last joined the session
		if !u.deadline.Stop() {
			<-u.deadline.C
		}
	}

	game, _ := s.SessionGame(u.sid)
	joined := u.m.makeMsgJoined(ctx, player.ID, u.sid, &game)
	u.m.sendToPlayer(player.tx, joined)

	gameStatus := u.m.makeMsgGameStatus(ctx, s.Players(u.sid))

	if reconnected {
		u.m.sendToPlayer(player.tx, gameStatus)
	} else {
		for _, tx := range s.PlayerTxs(u.sid) {
			u.m.sendToPlayer(tx, gameStatus)
		}
	}

	var stateMessage ServerTx
	switch state := s.sessionState(u.sid).(type) {
	case *AwaitingPlayersState:
		stateMessage = u.m.makeMsgWaiting(ctx, state.playersReady)
	case *GameStartedState:
		stateMessage = u.m.makeMsgGameStart(ctx, state.deadline)
	case *TaskStartedState:
		stateMessage = u.m.makeMsgTaskStart(ctx, state.taskIdx, state.deadline)
	case *PollStartedState:
		stateMessage = u.m.makeMsgPollStart(ctx, state.taskIdx, state.deadline, state.options)
	case *TaskEndedState:
		stateMessage = u.m.makeMsgTaskEnd(ctx, state.taskIdx, state.deadline, state.results)
	}
	u.m.sendToPlayer(player.tx, stateMessage)
}

func (u *sessionUpdater) removePlayer(ctx context.Context, s *UnsafeStorage, playerID PlayerID) {
	player, err := s.PlayerByID(u.sid, playerID)
	if err != nil {
		u.log.Printf("received removePlayer for unknown player: %s", err)
		return
	}

	if state, ok := s.sessionState(u.sid).(*AwaitingPlayersState); ok && state.owner == player.ClientID {
		// the owner left the session: close it.
		// note that we have to send an error to the owner too.
		// therefore we don't remove them here.
		for _, tx := range s.PlayerTxs(u.sid) {
			u.m.sendToPlayer(tx, u.m.makeMsgError(ctx, ErrOwnerLeft))
		}
		u.changeStateTo(ctx, s, nil)
		return
	}

	u.m.closePlayerTx(s, u.sid, playerID)
	s.removePlayer(u.sid, player.ClientID)

	gameStatus := u.m.makeMsgGameStatus(ctx, s.Players(u.sid))
	for _, tx := range s.PlayerTxs(u.sid) {
		u.m.sendToPlayer(tx, gameStatus)
	}

	switch state := s.sessionState(u.sid).(type) {
	case *AwaitingPlayersState:
		u.setPlayerStartReady(ctx, s, state, playerID, false)

	case *GameStartedState:
		// do nothing

	case *TaskStartedState:
		u.setPlayerAnswerReady(ctx, s, state, playerID, false)

	case *PollStartedState:
		u.setPlayerVote(ctx, s, state, playerID, NewOptionIdx(-1))

	case *TaskEndedState:
		// do nothing
	}
}

// changeStateTo changes the current session state to the nextState.
// If the nextState is nil, the session is closed.
func (u *sessionUpdater) changeStateTo(
	ctx context.Context,
	s *UnsafeStorage,
	nextState State,
) {
	if !u.deadline.Stop() {
		<-u.deadline.C
	}

	if nextState == nil {
		err := u.m.db.AcquireTx(ctx, func(tx pgx.Tx) error {
			u.m.closeSession(ctx, s, tx, u.sid)
			return tx.Commit(ctx)
		})
		if err != nil {
			u.log.Printf("could not close the session: %s", err)
		}

		return
	}

	// TODO: handle transition from other states

	u.deadline.Reset(nextState.Deadline().Sub(time.Now()))

	switch nextState.(type) {
	case *AwaitingPlayersState:
		// TODO

	case *GameStartedState:
		// TODO

	case *TaskStartedState:
		// TODO

	case *PollStartedState:
		// TODO

	case *TaskEndedState:
		// TODO
	}

	s.setSessionState(u.sid, nextState)
}

func (u *sessionUpdater) deadlineExpired(ctx context.Context, s *UnsafeStorage) {
	switch state := s.sessionState(u.sid).(type) {
	case *AwaitingPlayersState:
		for _, tx := range s.PlayerTxs(u.sid) {
			u.m.sendToPlayer(tx, u.m.makeMsgError(ctx, ErrNoOwnerTimeout))
		}

		u.changeStateTo(ctx, s, nil)

	case *GameStartedState:
		u.changeStateTo(ctx, s, u.makeFirstTaskStartedState(s, state))

	case *TaskStartedState:
		// TODO: go either to PollStartedState or TaskEndedState

	case *PollStartedState:
		u.changeStateTo(ctx, s, u.makePollTaskEndedState(s, state))

	case *TaskEndedState:
		// TODO: go either to TaskStartedState or finish the game
	}
}

func (u *sessionUpdater) setPlayerStartReady(
	ctx context.Context,
	s *UnsafeStorage,
	state *AwaitingPlayersState,
	playerID PlayerID,
	ready bool,
) {
	_, exists := state.playersReady[playerID]
	if ready == exists {
		return
	}

	if ready {
		state.playersReady[playerID] = struct{}{}
	} else {
		delete(state.playersReady, playerID)
	}

	waiting := u.m.makeMsgWaiting(ctx, state.playersReady)
	for _, tx := range s.PlayerTxs(u.sid) {
		u.m.sendToPlayer(tx, waiting)
	}

	if u.shouldStartGame(s) {
		u.changeStateTo(ctx, s, u.makeGameStartedState(s, state))
	}
}

func (u *sessionUpdater) shouldStartGame(s *UnsafeStorage) (start bool) {
	state, ok := s.sessionState(u.sid).(*AwaitingPlayersState)
	if !ok {
		return
	}

	owner, err := s.PlayerByClientID(u.sid, state.owner)
	if err != nil {
		return
	}
	if _, ok := state.playersReady[owner.ID]; !ok {
		return
	}

	if state.requireReady {
		for _, player := range s.Players(u.sid) {
			if _, ok := state.playersReady[player.ID]; !ok {
				return
			}
		}
	}

	return true
}

func (u *sessionUpdater) setPlayerAnswerReady(
	ctx context.Context,
	s *UnsafeStorage,
	state *TaskStartedState,
	playerID PlayerID,
	ready bool,
) {
	// TODO
}

func (u *sessionUpdater) setPlayerVote(
	ctx context.Context,
	s *UnsafeStorage,
	state *PollStartedState,
	playerID PlayerID,
	vote OptionIdx,
) {
	// TODO
}

func (u *sessionUpdater) makeGameStartedState(s *UnsafeStorage, state *AwaitingPlayersState) *GameStartedState {
	return &GameStartedState{
		deadline: time.Now().Add(GameStartedTimeout),
	}
}

func (u *sessionUpdater) makeFirstTaskStartedState(s *UnsafeStorage, state *GameStartedState) *TaskStartedState {
	// TODO
	return nil
}

func (u *sessionUpdater) makeNextTaskStartedState(s *UnsafeStorage, state *TaskEndedState) *TaskStartedState {
	// TODO
	return nil
}

func (u *sessionUpdater) makePollStartedState(s *UnsafeStorage, state *TaskStartedState) *PollStartedState {
	// TODO
	return nil
}

func (u *sessionUpdater) makePlainTaskEndedState(s *UnsafeStorage, state *TaskStartedState) *TaskEndedState {
	// TODO
	return nil
}

func (u *sessionUpdater) makePollTaskEndedState(s *UnsafeStorage, state *PollStartedState) *TaskEndedState {
	// TODO
	return nil
}
