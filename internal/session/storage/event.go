package storage

import "context"

type TxChan chan<- ServerTx

type ServerTx interface {
	Context() context.Context
	isServerTx()
}

type baseTx struct {
	Context context.Context
}

type MsgError struct {
	baseTx

	Inner error
}

func (*MsgError) isServerTx() {}

type MsgJoined struct {
	PlayerId  PlayerId
	SessionId SessionId
	Game      *Game
}

func (*MsgJoined) isServerTx() {}
