package session

import "context"

// # Update messages

type updateMsg interface {
	isUpdateMsg()
}

type updateMsgPlayerAdded struct {
	playerId PlayerId
}

func (*updateMsgPlayerAdded) isUpdateMsg() {}

type updateMsgChangeStateTo struct {
	nextState State
}

func (updateMsgChangeStateTo) isUpdateMsg() {}

// # Run logic

type sessionUpdater struct {
	m   *Manager
	sid SessionId
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

func (u *sessionUpdater) changeStateTo(nextState State) {
	// TODO: kill tickers, spawn tickers, etc
}
