package session

import "time"

var (
	NoOwnerTimeout     = 5 * time.Minute
	GameStartedTimeout = 5 * time.Second
	TaskEndTimeout     = 10 * time.Second
)

// How many points players gain for correctly answering questions.
var (
	CheckedTextTaskPoints Score = 2
	ChoiceTaskPoints      Score = 2
)
