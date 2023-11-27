package session

import "errors"

var (
	ErrNoOwnerTimeout = errors.New("timed out waiting for the owner to join")
)

var (
	ErrNoSession      = errors.New("no such session")
	ErrGameInProgress = errors.New("game has already started")
	ErrClientBanned   = errors.New("client was banned by the op")
	ErrNicknameUsed   = errors.New("nickname is in use")
	ErrLobbyFull      = errors.New("lobby is full")
)
