package session

import (
	"context"
	"time"
)

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

type MsgTaskStart struct {
	baseTx

	TaskIdx  int
	Deadline time.Time

	// Options must be only for ChoiceTask otherwise must be nil
	Options *[]string

	// ImgID must be only for PhotoTask otherwise must be nil
	ImgID *ImageID
}

func (*MsgTaskStart) isServerTx() {}

type EventAnswer interface {
	isEventAnswer()
}

type ChoiceEventAnswer string
type PhotoEventAnswer ImageID
type TextEventAnswer string
type CheckedTextEventAnswer string

func (*ChoiceEventAnswer) isEventAnswer()      {}
func (*PhotoEventAnswer) isEventAnswer()       {}
func (*TextEventAnswer) isEventAnswer()        {}
func (*CheckedTextEventAnswer) isEventAnswer() {}

type EventTaskAnswer struct {
	Value       EventAnswer
	PlayerCount uint16
	Correct     bool
}

type MsgTaskEnd struct {
	baseTx

	TaskIdx  int
	Deadline time.Time

	Answers []EventTaskAnswer

	// TODO: scoreboard
}

func (*MsgTaskEnd) isServerTx() {}
