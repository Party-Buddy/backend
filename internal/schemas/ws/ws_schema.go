package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas"
	"party-buddy/internal/util"
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

type MessageID uint32

func (id MessageID) String() string {
	return fmt.Sprint(uint32(id))
}

// A Time wraps a time.Time to provide a session protocol-compliant JSON encoding.
// Its value is represented as a Unix timestamp (a 64-bit integer).
type Time time.Time

func (t *Time) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte{}, errors.New("failed to serialize data because Time is nil")
	}
	return json.Marshal(time.Time(*t).UnixMilli())
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var val int64
	err := json.Unmarshal(data, &val)
	if err != nil {
		return err
	}
	*t = Time(time.UnixMilli(val))
	return nil
}

type BaseMessage struct {
	MsgID *MessageID   `json:"msg-id"`
	Kind  *MessageKind `json:"kind"`
	Time  *Time        `json:"time"`
}

func (m *BaseMessage) GetKind() MessageKind {
	return *m.Kind
}

func (m *BaseMessage) GetMsgID() MessageID {
	return *m.MsgID
}

func (m *BaseMessage) SetMsgID(id MessageID) {
	m.MsgID = &id
}

func (m *BaseMessage) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	return f.
		Is(validate.FieldValue(m.MsgID, "msg-id", "msg-id").Set()).
		Is(validate.FieldValue(m.Kind, "kind", "kind").Set()).
		Is(validate.FieldValue(m.Time, "time", "time").Set())
}

// A RecvMessage is implemented by protocol messages that can be received from a client.
type RecvMessage interface {
	validate.Validator

	GetKind() MessageKind
	GetMsgID() MessageID

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
	ErrReconnected    ErrorKind = "reconnected"
)

// JoinErrorKind codes
var (
	ErrSessionExpired ErrorKind = "session-expired"
	ErrLobbyFull      ErrorKind = "lobby-full"
	ErrNicknameUsed   ErrorKind = "nickname-used"
	ErrUnknownSession ErrorKind = "unknown-session"
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
	RefID   *MessageID `json:"ref-id"`
	Code    ErrorKind  `json:"code"`
	Message string     `json:"message"`
}

func (e *Error) Error() string {
	if e.RefID == nil {
		return fmt.Sprintf("%v (code `%v`)", e.Message, e.Code)
	} else {
		return fmt.Sprintf("%v (code `%v`, reply to msg %v)", e.Message, e.Code, e.RefID)
	}
}

func (e *Error) String() string {
	return e.Message
}

type MessageError struct {
	BaseMessage
	Error
}

func (*MessageError) isRespMessage() {}

func (*MessageError) isRecvMessage() {}

type MessageJoin struct {
	BaseMessage

	Nickname *string `json:"nickname"`
}

var nicknameRegex *regexp.Regexp = regexp.MustCompile("^[a-zA-Zа-яА-Я._ 0-9]{1,20}$")

func (m *MessageJoin) Validate(ctx context.Context) *valgo.Validation {
	return m.BaseMessage.Validate(ctx).
		Is(validate.FieldValue(m.Nickname, "nickname", "nickname").Set()).
		Is(valgo.StringP(m.Nickname, "nickname", "nickname").MatchingTo(nicknameRegex, "{{title}} is invalid")).
		Is(valgo.StringP(m.Kind, "kind", "kind").EqualTo(MsgKindJoin))
}

type MessageReady struct {
	BaseMessage

	Ready *bool `json:"ready"`
}

func (m *MessageReady) Validate(ctx context.Context) *valgo.Validation {
	return m.BaseMessage.Validate(ctx).
		Is(validate.FieldValue(m.Ready, "ready", "ready").Set()).
		Is(valgo.StringP(m.Kind, "kind", "kind").EqualTo(MsgKindReady))
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

type RecvAnswerType string

var validRecvAnswerTypes = []RecvAnswerType{CheckedText, Text, Option}

const (
	CheckedText RecvAnswerType = "checked-text"
	Text        RecvAnswerType = "text"
	Option      RecvAnswerType = "option"
)

type RecvAnswer struct {
	Type   *RecvAnswerType
	Option *uint8
	Text   *string
}

func (a *RecvAnswer) UnmarshalJSON(data []byte) error {
	var base struct {
		Type *RecvAnswerType `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	a.Type = base.Type

	if a.Type == nil {
		return nil
	}

	switch *a.Type {
	case Text, CheckedText:
		var answer struct {
			Value *string `json:"value"`
		}
		if err := json.Unmarshal(data, &answer); err != nil {
			return err
		}
		a.Text = answer.Value

	case Option:
		var answer struct {
			Value *uint8 `json:"value"`
		}
		if err := json.Unmarshal(data, &answer); err != nil {
			return err
		}
		a.Option = answer.Value
	}

	return nil
}

func (a *RecvAnswer) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	v := f.Is(valgo.StringP(a.Type, "type", "type").Not().Nil().InSlice(validRecvAnswerTypes))
	if a.Type == nil {
		return v
	}
	switch *a.Type {
	case Option:
		v.Is(valgo.Uint8P(a.Option, "value", "value").Not().Nil().
			LessThan(configuration.OptionsCount))
	case Text:
		v.Is(valgo.StringP(a.Text, "value", "value").Not().Nil().
			MatchingTo(configuration.BaseTextReg).
			Passing(util.MaxLengthPChecker(configuration.MaxTextAnswerLength)))
	case CheckedText:
		v.Is(valgo.StringP(a.Text, "value", "value").Not().Nil().
			MatchingTo(configuration.CheckedTextAnswerReg).
			Passing(util.MaxLengthPChecker(configuration.MaxCheckedTextAnswerLength)))
	}
	return v
}

type MessageTaskAnswer struct {
	BaseMessage

	TaskIdx *int        `json:"task-idx"`
	Ready   *bool       `json:"ready"`
	Answer  *RecvAnswer `json:"answer,omitempty"`
}

func (m *MessageTaskAnswer) Validate(ctx context.Context) *valgo.Validation {
	v := m.BaseMessage.Validate(ctx).
		Is(valgo.IntP(m.TaskIdx, "task-idx", "task-idx").Not().Nil().LessThan(configuration.MaxTaskCount)).
		Is(valgo.BoolP(m.Ready, "ready", "ready").Not().Nil()).
		Is(valgo.StringP(m.Kind, "kind", "kind").Not().Nil().EqualTo(MsgKindTaskAnswer))
	if m.Answer != nil {
		v.Merge(m.Answer.Validate(ctx))
	}
	return v
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
	refID MessageID
	kind  MessageKind
}

func (e *UnknownMessageError) RefID() MessageID {
	return e.refID
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
	refID MessageID
	cause *valgo.Error
}

func (e *ValidationError) RefID() MessageID {
	return e.refID
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
			refID: *base.MsgID,
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
			refID: *base.MsgID,
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
			RefID:   nil,
			Code:    ErrMalformedMsg,
			Message: fmt.Sprintf("in field `%s`: %s has an illegal type", typeError.Field, typeError.Value),
		}
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return &Error{
			RefID:   nil,
			Code:    ErrMalformedMsg,
			Message: fmt.Sprintf("syntax error %s", syntaxError),
		}
	}

	var decodeError *DecodeError
	if errors.As(err, &decodeError) {
		return &Error{
			RefID:   nil,
			Code:    ErrMalformedMsg,
			Message: fmt.Sprintf("message is not valid JSON: %s", decodeError),
		}
	}

	var unknownMessageError *UnknownMessageError
	if errors.As(err, &unknownMessageError) {
		refID := unknownMessageError.RefID()
		return &Error{
			RefID:   &refID,
			Code:    ErrProtoViolation,
			Message: fmt.Sprintf("unacceptable message kind: `%s`", unknownMessageError.Kind()),
		}
	}

	var validationError *ValidationError
	if errors.As(err, &validationError) {
		refID := validationError.RefID()
		message := "malformed message"
		if fieldName, msg, ok := validate.ExtractValgoErrorFields(validationError.ValgoError()); ok {
			message = fmt.Sprintf("in field `%s`: %s", fieldName, msg)
		}

		return &Error{
			RefID:   &refID,
			Code:    ErrMalformedMsg,
			Message: message,
		}
	}

	return &Error{
		RefID:   nil,
		Code:    ErrInternal,
		Message: "internal failure while decoding message",
	}
}

type RespMessage interface {
	isRespMessage()

	GetKind() MessageKind
	SetMsgID(id MessageID)
}

type MessageJoined struct {
	BaseMessage

	RefID      *MessageID          `json:"ref-id"`
	PlayerID   uint32              `json:"player-id"`
	Sid        uuid.UUID           `json:"session-id"`
	Game       schemas.GameDetails `json:"game"`
	MaxPlayers uint8               `json:"max-players"`
}

func (*MessageJoined) isRespMessage() {}

type Player struct {
	PlayerID uint32 `json:"player-id"`
	Nickname string `json:"nickname"`
}

type MessageGameStatus struct {
	BaseMessage

	Players []Player `json:"players"`
}

func (*MessageGameStatus) isRespMessage() {}

type MessageTaskStart struct {
	BaseMessage

	TaskIdx  uint8     `json:"task-idx"`
	Deadline time.Time `json:"deadline"`

	Options *[]string `json:"options,omitempty"`

	ImgURI *string `json:"img-uri,omitempty"`
}

func (*MessageTaskStart) isRespMessage() {}

type Answer interface {
	isAnswer()
}

type CheckedWordAnswer struct {
	// Value of answer for the checked text task
	Value       string `json:"value"`
	PlayerCount uint16 `json:"player-count"`
	Correct     bool   `json:"correct"`
}

func (*CheckedWordAnswer) isAnswer() {}

type PhotoAnswer struct {
	// Value should be an image uri
	Value string `json:"value"`
	Votes uint16 `json:"votes"`
}

func (*PhotoAnswer) isAnswer() {}

type WordAnswer struct {
	// Value of answer for text task
	Value string `json:"value"`
	Votes uint16 `json:"votes"`
}

func (*WordAnswer) isAnswer() {}

type TaskOptionAnswer struct {
	// Value should be the one option from choice task
	Value       string `json:"value"`
	PlayerCount uint16 `json:"player-count"`
	Correct     bool   `json:"correct"`
}

func (*TaskOptionAnswer) isAnswer() {}

type TaskPlayerScore struct {
	PlayerID    uint32 `json:"player-id"`
	TaskPoints  uint32 `json:"task-points"`
	TotalPoints uint32 `json:"total-points"`
}

type MessageTaskEnd struct {
	BaseMessage

	TaskIdx    uint8             `json:"task-idx"`
	Deadline   time.Time         `json:"deadline"`
	Scoreboard []TaskPlayerScore `json:"scoreboard"`
	Answers    []Answer          `json:"answers"`
}

func (*MessageTaskEnd) isRespMessage() {}

type MessageGameStart struct {
	BaseMessage

	Deadline time.Time `json:"deadline"`
}

func (*MessageGameStart) isRespMessage() {}

type MessageWaiting struct {
	BaseMessage

	Ready []uint32 `json:"ready"`
}

func (*MessageWaiting) isRespMessage() {}

type GamePlayerScore struct {
	PlayerID    uint32 `json:"player-id"`
	TotalPoints uint32 `json:"total-points"`
}

type MessageGameEnd struct {
	BaseMessage

	Scoreboard []GamePlayerScore `json:"scoreboard"`
}

func (*MessageGameEnd) isRespMessage() {}
