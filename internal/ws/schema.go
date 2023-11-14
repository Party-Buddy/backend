package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cohesivestack/valgo"
	"party-buddy/internal/validate"
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
	MsgId *MessageId   `json:"msg-id"`
	Kind  *MessageKind `json:"kind"`
	Time  *Time        `json:"time"`
}

func (m *BaseMessage) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	return f.
		Is(validate.FieldValue(m.MsgId, "msg-id", "msg-id").Set()).
		Is(validate.FieldValue(m.Kind, "kind", "kind").Set()).
		Is(validate.FieldValue(m.Time, "time", "time").Set())
}

// A RecvMessage is implemented by protocol messages that can be received from a client.
type RecvMessage interface {
	validate.Validator
	isRecvMessage()
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

type MessageJoin struct {
	BaseMessage

	// TODO
}

func (m *MessageJoin) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

type MessageReady struct {
	BaseMessage

	// TODO
}

func (m *MessageReady) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

type MessageKick struct {
	BaseMessage

	// TODO
}

func (m *MessageKick) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

type MessageLeave struct {
	BaseMessage

	// TODO
}

func (m *MessageLeave) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

type MessageTaskAnswer struct {
	BaseMessage

	// TODO
}

func (m *MessageTaskAnswer) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

type MessagePollChoose struct {
	BaseMessage

	// TODO
}

func (m *MessagePollChoose) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO
	return f.New()
}

func (*MessageJoin) isRecvMessage()       {}
func (*MessageReady) isRecvMessage()      {}
func (*MessageKick) isRecvMessage()       {}
func (*MessageLeave) isRecvMessage()      {}
func (*MessageTaskAnswer) isRecvMessage() {}
func (*MessagePollChoose) isRecvMessage() {}

type UnknownMessageError struct {
	kind MessageKind
}

func (e *UnknownMessageError) Error() string {
	return fmt.Sprintf("unknown message kind `%v`", e.kind)
}

type DecodeError struct {
	cause error
}

func (e *DecodeError) Error() string {
	return e.cause.Error()
}

func (e *DecodeError) Unwrap() error {
	return e.cause
}

// ParseMessage decodes and validates a protocol message.
//
// If the supplied data is invalid, returns one of the following errors:
// - [DecodeError] if some an error has occurred during decoding
// - [UnknownMessageError] if the message data specifies an unknown message kind
// - [valgo.Error] if validation fails
// - or possibly some other error type.
func ParseMessage(ctx context.Context, data []byte) (RecvMessage, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}
	if val := base.Validate(ctx); !val.Valid() {
		return nil, val.Error()
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var msg RecvMessage

	switch *base.Kind {
	case MsgKindJoin:
		msg = &MessageJoin{}
	case MsgKindReady:
		msg = &MessageReady{}
	case MsgKindKick:
		msg = &MessageKick{}
	case MsgKindLeave:
		msg = &MessageLeave{}
	case MsgKindTaskAnswer:
		msg = &MessageTaskAnswer{}
	case MsgKindPollChoose:
		msg = &MessagePollChoose{}
	default:
		return nil, &UnknownMessageError{kind: *base.Kind}
	}

	if err := decoder.Decode(msg); err != nil {
		return nil, &DecodeError{cause: err}
	}

	if val := msg.Validate(ctx); !val.Valid() {
		return nil, val.Error()
	}

	return msg, nil
}
