package session

import (
	"context"
	"party-buddy/internal/session/storage"
)

// # Update messages

type updateMsg interface {
	isUpdateMsg()
}

type updateMsgPlayerAdded struct {
	playerId storage.PlayerId
}

func (*updateMsgPlayerAdded) isUpdateMsg() {}

type updateMsgChangeStateTo struct {
	nextState storage.State
}

func (updateMsgChangeStateTo) isUpdateMsg() {}

// # Run logic

type sessionUpdater struct {
	m   *Manager
	sid storage.SessionId
	rx  <-chan updateMsg
}

func (u *sessionUpdater) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case msg := <-u.rx:
			switch msg := msg.(type) {
			case *updateMsgPlayerAdded:
				// TODO: do something?
			case *updateMsgChangeStateTo:
				u.changeStateTo(msg.nextState)
			}
		}
	}
}

func (u *sessionUpdater) changeStateTo(nextState storage.State) {
	// TODO: kill tickers, spawn tickers, etc
}
