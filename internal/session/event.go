package session

import "context"

type TxChan chan<- ServerTx

// A ServerTx is a message sent from the server to the client.
//
// Once the message is sent, its fields must not be updated.
type ServerTx interface {
	Context() context.Context
	isServerTx()
}

type baseTx struct {
	Ctx context.Context
}

func (m *baseTx) Context() context.Context {
	return m.Ctx
}

type MsgError struct {
	baseTx

	Inner error
}

func (*MsgError) isServerTx() {}

type MsgJoined struct {
	baseTx
	PlayerID  PlayerID
	SessionID SessionID
	Game      *Game
}

func (*MsgJoined) isServerTx() {}
