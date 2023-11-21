package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"party-buddy/internal/schemas"
	"party-buddy/internal/validate"
	"regexp"
	"time"

	"github.com/cohesivestack/valgo"
)

// See internal/api/api_schema.go for information on serialization/deserialization.
// Use [ParseMessage] for message deserialization as well as [ParseErrorToMessageError] for error handling.

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

func (*BaseMessage) isRespMessage() {}

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

	Nickname *string `json:"nickname"`
}

var nicknameRegex *regexp.Regexp = regexp.MustCompile("^[a-zA-Zа-яА-Я._ 0-9]{1,20}$")

func (m *MessageJoin) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	return f.Is(validate.FieldValue(m.Nickname, "nickname", "nickname").Set()).
		Is(valgo.StringP(m.Nickname, "nickname", "nickname").MatchingTo(nicknameRegex, "{{title}} is invalid")).
		Is(valgo.StringP(m.Kind, "kind", "kind").EqualTo(MsgKindJoin))
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
	refId MessageId
	kind  MessageKind
}

func (e *UnknownMessageError) RefId() MessageId {
	return e.refId
}

func (e *UnknownMessageError) Kind() MessageKind {
	return e.kind
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

type ValidationError struct {
	refId MessageId
	cause *valgo.Error
}

func (e *ValidationError) RefId() MessageId {
	return e.refId
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s", e.cause)
}

func (e *ValidationError) ValgoError() *valgo.Error {
	return e.cause
}

func (e *ValidationError) Unwrap() error {
	return e.cause
}

// ParseMessage decodes and validates a protocol message.
//
// If the supplied data is invalid, returns one of the following errors:
// - [DecodeError] if some an error has occurred during decoding
// - [UnknownMessageError] if the message data specifies an unknown message kind
// - [ValidationError] if validation fails
// - or possibly some other error type.
func ParseMessage(ctx context.Context, data []byte) (RecvMessage, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}
	if val := base.Validate(ctx); !val.Valid() {
		return nil, &ValidationError{
			refId: *base.MsgId,
			cause: val.Error().(*valgo.Error),
		}
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
		return nil, &ValidationError{
			refId: *base.MsgId,
			cause: val.Error().(*valgo.Error),
		}
	}

	return msg, nil
}

// ParseErrorToMessageError converts an error returned by [ParseMessage] to an [Error] message.
func ParseErrorToMessageError(err error) error {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return &Error{
			RefId:   nil,
			Code:    ErrMalformedMsg,
			Message: fmt.Sprintf("in field `%s`: %s has an illegal type", typeError.Field, typeError.Value),
		}
	}

	var decodeError *DecodeError
	if errors.As(err, &decodeError) {
		return &Error{
			RefId:   nil,
			Code:    ErrMalformedMsg,
			Message: fmt.Sprintf("message is not valid JSON: %s", decodeError),
		}
	}

	var unknownMessageError *UnknownMessageError
	if errors.As(err, &unknownMessageError) {
		refId := unknownMessageError.RefId()
		return &Error{
			RefId:   &refId,
			Code:    ErrProtoViolation,
			Message: fmt.Sprintf("unacceptable message kind: `%s`", unknownMessageError.Kind()),
		}
	}

	var validationError *ValidationError
	if errors.As(err, &validationError) {
		refId := validationError.RefId()
		message := "malformed message"
		if fieldName, msg, ok := validate.ExtractValgoErrorFields(validationError.ValgoError()); ok {
			message = fmt.Sprintf("in field `%s`: %s", fieldName, msg)
		}

		return &Error{
			RefId:   &refId,
			Code:    ErrMalformedMsg,
			Message: message,
		}
	}

	return &Error{
		RefId:   nil,
		Code:    ErrInternal,
		Message: "internal failure while decoding message",
	}
}

type RespMessage interface {
	isRespMessage()
}

type MessageJoined struct {
	BaseMessage

	RefID    *MessageId          `json:"ref-id"`
	PlayerID uint32              `json:"player-id"`
	Sid      uuid.UUID           `json:"session-id"`
	Game     schemas.GameDetails `json:"game"`
}

func (*MessageJoined) isRespMessage() {}
