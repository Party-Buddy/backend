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
	playerID PlayerID
}

func (*updateMsgPlayerAdded) isUpdateMsg() {}

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
	deadline time.Timer
}

func (u *sessionUpdater) run(ctx context.Context) error {
	// TODO: set the deadline for the current state somehow (changeStateTo?)

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
					u.playerAdded(s, msg.playerID)
				case *updateMsgChangeStateTo:
					u.changeStateTo(ctx, s, msg.nextState)
				}
			})
		}
	}
}

func (u *sessionUpdater) playerAdded(s *UnsafeStorage, playerID PlayerID) {
	player, err := s.PlayerByID(u.sid, playerID)
	if err != nil {
		u.log.Printf("while handling added player: %s", err)
		return
	}

	state := s.sessionState(u.sid)
	if state == nil {
		return
	}

	switch state := state.(type) {
	case *AwaitingPlayersState:
		if state.owner == player.ClientID {
			// the owner has at last joined the session
			if !u.deadline.Stop() {
				<-u.deadline.C
			}
		}
	default:
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

	// TODO: handle transition to other states
}

func (u *sessionUpdater) deadlineExpired(ctx context.Context, s *UnsafeStorage) {
	switch s.sessionState(u.sid).(type) {
	case *AwaitingPlayersState:
		for _, tx := range s.PlayerTxs(u.sid) {
			u.m.sendToPlayer(tx, &MsgError{
				baseTx: baseTx{
					Ctx: ctx,
				},
				Inner: ErrNoOwnerTimeout,
			})
		}

		u.changeStateTo(ctx, s, nil)
	}
}
