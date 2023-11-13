package ws

import (
	"encoding/json"
	"fmt"
	"time"
)

type MessageKind string

var (
	MsgKindError      MessageKind = "error"
	MsgKindJoin       MessageKind = "join"
	MsgKindJoined     MessageKind = "joined"
	MsgKindGameStatus MessageKind = "game-status"
	MsgKindReady      MessageKind = "ready"
	MsgKindKick       MessageKind = "kick"
	MsgKindLeave      MessageKind = "leave"
	MsgKindTaskStart  MessageKind = "task-start"
	MsgKindTaskAnswer MessageKind = "task-answer"
	MsgKindPollStart  MessageKind = "poll-start"
	MsgKindPollChoose MessageKind = "poll-choose"
	MsgKindTaskEnd    MessageKind = "task-end"
	MsgKindGameEnd    MessageKind = "game-end"
	MsgKindGameStart  MessageKind = "game-start"
	MsgKindWaiting    MessageKind = "waiting"
)

func (m MessageKind) MarshalText() ([]byte, error) {
	return []byte(m), nil
}

type MessageId uint32

func (id MessageId) String() string {
	return fmt.Sprint(uint32(id))
}

// A Time wraps a time.Time to provide a session protocol-compliant JSON encoding.
// Its value is represented as a Unix timestamp (a 64-bit integer).
type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UnixMilli())
}

type BaseMessage struct {
	MsgId MessageId   `json:"msg-id"`
	Kind  MessageKind `json:"kind"`
	Time  Time        `json:"time"`
}

type ErrorKind string

func (e ErrorKind) MarshalText() ([]byte, error) {
	return []byte(e), nil
}

// GeneralErrorKind codes
var (
	ErrInternal       ErrorKind = "internal"
	ErrMalformedMsg   ErrorKind = "malformed-msg"
	ErrProtoViolation ErrorKind = "proto-violation"
)

// JoinErrorKind codes
var (
	ErrSessionExpired ErrorKind = "session-expired"
	ErrLobbyFull      ErrorKind = "lobby-full"
	ErrNicknameUsed   ErrorKind = "nickname-used"
)

// OpErrorKind codes
var (
	ErrOpOnly ErrorKind = "op-only"
)

// GameErrorKind codes
var (
	ErrInactivity    ErrorKind = "inactivity"
	ErrSessionClosed ErrorKind = "session-closed"
)

type Error struct {
	RefId   *MessageId `json:"ref-id"`
	Code    ErrorKind  `json:"code"`
	Message string     `json:"message"`
}

func (e *Error) Error() string {
	if e.RefId == nil {
		return fmt.Sprintf("%v (code `%v`)", e.Message, e.Code)
	} else {
		return fmt.Sprintf("%v (code `%v`, reply to msg %v)", e.Message, e.Code, e.RefId)
	}
}

func (e *Error) String() string {
	return e.Message
}

type MessageError struct {
	BaseMessage
	Error
}
