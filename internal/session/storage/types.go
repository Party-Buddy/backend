package storage

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
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

func NewSessionId() SessionId {
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("could not generate session id: %v", err))
	}
	return SessionId(uuid)
}

func NewPlayerId() PlayerId {
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("could not generate player id: %v", err))
	}
	return PlayerId(uuid)
}

// An InviteCode is a short code used for session discovery.
// Only valid until the game starts.
type InviteCode string

func NewInviteCode() InviteCode {
	// We need to generate an arbitrary invite code satisfying [A-Z0-9]{6}.
	// There are 36 possibilities for each character.
	var builder strings.Builder
	maxN := big.NewInt(36)

	for i := 0; i < 6; i++ {
		r, err := rand.Int(rand.Reader, maxN)
		if err != nil {
			panic(fmt.Sprintf("could not generate invite code: %v", err))
		}

		switch n := r.Uint64(); {
		case n < 10:
			builder.WriteByte(byte('0' + n)) // n ∈ [0, 9]
		default:
			builder.WriteByte(byte('A' + n - 10)) // n ∈ [10, 35]
		}
	}

	return InviteCode(builder.String())
}

type session struct {
	id            SessionId
	game          Game
	players       map[PlayerId]Player
	playersMax    int
	clients       map[ClientId]PlayerId
	bannedClients map[ClientId]struct{}
	state         state
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

	// TODO:
	// // Connection holds a channel that allows sending events to the client.
	// Connection chan<- event.ServerEvent
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
