package session

import "errors"

// NOTE: if you're adding an error here, don't forget to fix the switch in ws/converters/error

var (
	ErrNoOwnerTimeout = errors.New("timed out waiting for the owner to join")
	ErrReconnected    = errors.New("client joined the session from another connection")
	ErrOwnerLeft      = errors.New("owner left the session")
)

var (
	ErrNoSession      = errors.New("no such session")
	ErrGameInProgress = errors.New("game has already started")
	ErrClientBanned   = errors.New("client was banned by the op")
	ErrNicknameUsed   = errors.New("nickname is in use")
	ErrLobbyFull      = errors.New("lobby is full")
)

var (
	ErrInternal = errors.New("internal error")
	ErrNoPlayer = errors.New("no player with such id")
)

var (
	ErrTaskNotStartedYet          = errors.New("task hasn't been started yet")
	ErrTypesTaskAndAnswerMismatch = errors.New("answer type cannot be used with this task")
	ErrTaskIndexOutOfBounds       = errors.New("no task with such index")
)
