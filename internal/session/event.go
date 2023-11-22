package session

import "context"

type TxChan chan<- ServerTx

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
	PlayerId  PlayerId
	SessionId SessionId
	Game      *Game
}

func (*MsgJoined) isServerTx() {}
