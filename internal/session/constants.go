package session

import "time"

var (
	NoOwnerTimeout     = time.Duration(5) * time.Minute
	GameStartedTimeout = time.Duration(5) * time.Second
)
