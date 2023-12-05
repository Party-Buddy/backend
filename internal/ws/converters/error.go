package converters

import (
	"errors"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
)

// ErrorKindAndMessage takes a session package error and returns
// its error kind and message suitable for sending over the wire.
func ErrorKindAndMessage(err error) (kind ws.ErrorKind, msg string) {
	switch {
	case errors.Is(err, session.ErrNoOwnerTimeout):
		return ws.ErrSessionClosed, "timed out waiting for the owner"
	case errors.Is(err, session.ErrReconnected):
		return ws.ErrReconnected, "reconnected from another connection"
	case errors.Is(err, session.ErrOwnerLeft):
		return ws.ErrSessionClosed, "the owner left the session"
	case errors.Is(err, session.ErrNoSession), errors.Is(err, session.ErrGameInProgress), errors.Is(err, session.ErrClientBanned):
		return ws.ErrSessionExpired, "no such session"
	case errors.Is(err, session.ErrNicknameUsed):
		return ws.ErrNicknameUsed, "the nickname is already taken"
	case errors.Is(err, session.ErrLobbyFull):
		return ws.ErrLobbyFull, "the lobby has reached its maximum capacity"
	case errors.Is(err, session.ErrTaskNotStartedYet):
		return ws.ErrMalformedMsg, "the task hasn't been started yet"
	case errors.Is(err, session.ErrTypesTaskAndAnswerMismatch):
		return ws.ErrMalformedMsg, "the provided answer type cannot be used for this task"
	case errors.Is(err, session.ErrTaskIndexOutOfBounds):
		return ws.ErrMalformedMsg, "the task index is out of bounds"
	case errors.Is(err, session.ErrNoPlayer):
		return ws.ErrProtoViolation, "no such player in the session"
	default:
		return ws.ErrInternal, "an internal error has occured"
	}
}
