package storage

import (
	"time"

	"github.com/google/uuid"
)

type (
	// The ImageId identifies a particular image stored on this server.
	ImageId   string
	SessionId uuid.UUID
	PlayerId  uuid.UUID
	ClientId  uuid.UUID
)

// An InviteCode is a short code used for session discovery.
// Only valid until the game starts.
type InviteCode string

type session struct {
	id             SessionId
	game           Game
	players        map[PlayerId]Player
	clients        map[ClientId]PlayerId
	removedClients map[ClientId]struct{}
	state          state
}

type Game struct {
	Name        string
	Description string
	ImageId     ImageId
	DateChanged time.Time
	Tasks       []Task
}

type Player struct {
	Id       PlayerId
	ClientId ClientId
	Nickname string
}

type PollOption struct {
	Value TaskAnswer

	// Beneficiaries is a set of players who have submitted this answer and would benefit from having this option win.
	Beneficiaries map[PlayerId]struct{}
}

// An OptionIdx is a "nullable" index for an option in a poll.
// An invalid `OptionIdx` (its zero value) indicates a player has not selected an option yet.
type OptionIdx int

// NewOptionsIdx returns a new `OptionIdx`.
// It will be valid if `idx >= 0 && idx < INT_MAX`.
func NewOptionIdx(idx int) OptionIdx {
	if r := OptionIdx(idx + 1); int(r) > 0 {
		return r
	}

	return OptionIdx(0)
}

func (i OptionIdx) Valid() bool {
	return int(i) != 0
}

// Index returns the 0-based index corresponding to this `OptionIdx`.
// It only makes sense if the `OptionIdx` is valid.
func (i OptionIdx) Index() int {
	return int(i) - 1
}

type AnswerResult struct {
	Value TaskAnswer

	// Submissions is the number of players who have submitted this answer.
	Submissions int

	// Votes is the number of players who have voted for this answer.
	// (If there was no poll, this count is zero.)
	Votes int
}
