package session

import "errors"

var (
	ErrNoSession      = errors.New("no such session")
	ErrGameInProgress = errors.New("game has already started")
	ErrClientBanned   = errors.New("client was banned by the op")
	ErrNicknameUsed   = errors.New("nickname is in use")
	ErrLobbyFull      = errors.New("lobby is full")
)
