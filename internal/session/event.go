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

// MsgDisconnect is used to notify client that it should stop
// all info about reason of disconnecting should with another ServerTx
// BEFORE MsgDisconnect
type MsgDisconnect struct {
	baseTx
}

func (*MsgDisconnect) isServerTx() {}
