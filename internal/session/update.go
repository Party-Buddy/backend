package session

import (
	"context"
	"fmt"
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

func (*updateMsgChangeStateTo) isUpdateMsg() {}

type updateMsgSetPlayerReady struct {
	ctx      context.Context
	playerID PlayerID
	ready    bool
}

func (*updateMsgSetPlayerReady) isUpdateMsg() {}

type updateMsgUpdTaskAnswer struct {
	ctx      context.Context
	playerID PlayerID
	answer   TaskAnswer
	ready    bool
	taskIdx  int
}

func (*updateMsgUpdTaskAnswer) isUpdateMsg() {}

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
				u.log.Println("the updater channel has been closed, stopping")
				return nil
			}

			u.log.Printf("handling %T", msg)

			u.m.storage.Atomically(func(s *UnsafeStorage) {
				switch msg := msg.(type) {
				case *updateMsgPlayerAdded:
					u.playerAdded(msg.ctx, s, msg.playerID, msg.reconnected)
				case *updateMsgRemovePlayer:
					u.removePlayer(msg.ctx, s, msg.playerID)
				case *updateMsgChangeStateTo:
					u.changeStateTo(ctx, s, msg.nextState)
				case *updateMsgSetPlayerReady:
					u.setPlayerReady(ctx, s, msg.playerID, msg.ready)
				case *updateMsgUpdTaskAnswer:
					u.updateAnswer(msg.ctx, s, msg.playerID, msg.answer, msg.ready, msg.taskIdx)
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
	state := s.sessionState(u.sid)
	if state == nil {
		return
	}

	player, err := s.PlayerByID(u.sid, playerID)
	if err != nil {
		u.log.Printf("could not handle added player: %s", err)
		return
	}

	if state, ok := state.(*AwaitingPlayersState); ok && state.owner == player.ClientID {
		u.log.Printf("the owner %s has joined the session", state.owner)

		// the owner has at last joined the session
		u.deadline.Stop()
	}

	session, _ := s.sessionByID(u.sid)
	game := session.game
	joined := u.m.makeMsgJoined(ctx, player.ID, u.sid, &game, session.playersMax)
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
		task := game.Tasks[state.taskIdx]
		switch task.(type) {
		case PhotoTask:
			answer, ok := state.answers[playerID]
			if !ok {
				u.log.Panicf("no image registered for player %s (nickname=%q, clientID=%s)",
					playerID, player.Nickname, player.ClientID)
			}
			stateMessage = u.m.makeMsgTaskStart(ctx, state.taskIdx, state.deadline, task, answer)

		default:
			stateMessage = u.m.makeMsgTaskStart(ctx, state.taskIdx, state.deadline, task, nil)
		}

	case *PollStartedState:
		stateMessage = u.m.makeMsgPollStart(ctx, state.taskIdx, state.deadline, state.options)

	case *TaskEndedState:
		stateMessage = u.m.makeMsgTaskEnd(
			ctx,
			state.taskIdx,
			state.deadline,
			s.taskByIdx(u.sid, state.taskIdx),
			s.SessionScoreboard(u.sid),
			state.winners,
			state.results,
		)
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

	// TODO: close the session if there aren't any players left

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
	u.deadline.Stop()

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

	u.log.Printf("switching state to %T", nextState)
	u.deadline.Reset(nextState.Deadline().Sub(time.Now()))

	switch nextState := nextState.(type) {
	case *AwaitingPlayersState:
		// do nothing

	case *GameStartedState:
		u.m.sendToAllPlayers(s, u.sid, u.m.makeMsgGameStart(ctx, state.deadline))

	case *TaskStartedState:
		task := s.taskByIdx(u.sid, nextState.taskIdx)
		if task == nil {
			u.log.Panicf("task %d not found", nextState.taskIdx)
		}
		switch task.(type) {
		case PhotoTask:
			err := u.m.db.AcquireTx(ctx, func(tx pgx.Tx) error {
				var err error
				s.ForEachPlayer(u.sid, func(p Player) {
					var img ImageID
					img, err = u.m.newImgMetadataForSession(ctx, tx, u.sid, p.ClientID)
					if err != nil {
						err = fmt.Errorf("could not register an image for player %s (nickname=%q, clientID=%s): %w",
							p.ID, p.Nickname, p.ClientID, err)
						return
					}
					nextState.answers[p.ID] = PhotoTaskAnswer(img)
				})
				if err != nil {
					return err
				}

				tx.Commit(ctx)
				return nil
			})
			if err != nil {
				u.log.Printf("could not start a PhotoTask: %s", err)
				u.m.sendErrorToAllPlayers(ctx, s, u.sid, ErrInternal)
				u.m.db.AcquireTx(ctx, func(tx pgx.Tx) error {
					u.m.closeSession(ctx, s, tx, u.sid)
					tx.Commit(ctx)
					return nil
				})
				return
			}
			s.ForEachPlayer(u.sid, func(p Player) {
				u.m.sendToPlayer(p.tx, u.m.makeMsgTaskStart(ctx, nextState.taskIdx, nextState.deadline, task, nextState.answers[p.ID]))
			})

		default:
			for _, tx := range s.PlayerTxs(u.sid) {
				u.m.sendToPlayer(tx, u.m.makeMsgTaskStart(ctx, nextState.taskIdx, nextState.deadline, task, nil))
			}
		}

	case *PollStartedState:
		// TODO

	case *TaskEndedState:
		s.incrementScores(u.sid, nextState.winners)
		u.m.sendToAllPlayers(s, u.sid, u.m.makeMsgTaskEnd(
			ctx,
			nextState.taskIdx,
			nextState.deadline,
			s.taskByIdx(u.sid, nextState.taskIdx),
			s.SessionScoreboard(u.sid),
			nextState.winners,
			nextState.results,
		))
	}

	s.setSessionState(u.sid, nextState)
}

func (u *sessionUpdater) setPlayerReady(
	ctx context.Context,
	s *UnsafeStorage,
	playerID PlayerID,
	ready bool,
) {
	state := s.sessionState(u.sid)
	if state == nil {
		return
	}

	if _, err := s.PlayerByID(u.sid, playerID); err != nil {
		u.log.Printf("could not set player readiness: %s", err)
		return
	}

	if state, ok := state.(*AwaitingPlayersState); ok {
		u.setPlayerStartReady(ctx, s, state, playerID, ready)
	}
}

func (u *sessionUpdater) updateAnswer(
	ctx context.Context,
	s *UnsafeStorage,
	playerID PlayerID,
	answer TaskAnswer,
	ready bool,
	taskIdx int,
) {
	state, ok := s.sessionState(u.sid).(*TaskStartedState)
	if !ok {
		return
	}

	player, err := s.PlayerByID(u.sid, playerID)
	if err != nil {
		u.log.Printf("could not update the answer for task %d: %s", taskIdx, err)
		return
	}

	switch {
	case state.taskIdx > taskIdx:
		return

	case state.taskIdx < taskIdx:
		u.m.sendToPlayer(player.tx, u.m.makeMsgError(ctx, ErrTaskNotStartedYet))
		u.m.closePlayerTx(s, u.sid, playerID)
		return
	}

	if answer != nil {
		state.answers[playerID] = answer
	}
	u.setPlayerAnswerReady(ctx, s, state, playerID, ready)
}

func (u *sessionUpdater) deadlineExpired(ctx context.Context, s *UnsafeStorage) {
	switch state := s.sessionState(u.sid).(type) {
	case *AwaitingPlayersState:
		u.m.sendErrorToAllPlayers(ctx, s, u.sid, ErrNoOwnerTimeout)
		u.changeStateTo(ctx, s, nil)

	case *GameStartedState:
		u.changeStateTo(ctx, s, u.makeFirstTaskStartedState(s, state))

	case *TaskStartedState:
		task := s.taskByIdx(u.sid, state.taskIdx)
		if task == nil {
			u.log.Panicf("task %d not found", state.taskIdx)
		}
		if task.NeedsPoll() {
			u.changeStateTo(ctx, s, u.makePollStartedState(s, state))
		} else {
			u.changeStateTo(ctx, s, u.makePlainTaskEndedState(s, state))
		}

	case *PollStartedState:
		u.changeStateTo(ctx, s, u.makePollTaskEndedState(s, state))

	case *TaskEndedState:
		if s.hasNextTask(u.sid, state.taskIdx) {
			u.changeStateTo(ctx, s, u.makeNextTaskStartedState(s, state))
		} else {
			u.finishGame(ctx, s, state)
		}
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
	if ready {
		state.ready[playerID] = struct{}{}
	} else {
		delete(state.ready, playerID)
	}
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

// finishGame finishes the game normally.
//
// If you're looking for a way to close a session quickly, just do changeStateTo(..., nil).
func (u *sessionUpdater) finishGame(ctx context.Context, s *UnsafeStorage, state *TaskEndedState) {
	if !s.SessionExists(u.sid) {
		// could've got closed already
		return
	}

	u.m.sendToAllPlayers(s, u.sid, u.m.makeMsgGameEnd(ctx, s.SessionScoreboard(u.sid)))
	u.changeStateTo(ctx, s, nil)
}
