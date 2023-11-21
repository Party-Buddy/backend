package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type (
	// An ImageId identifies a particular image stored on this server.
	ImageId uuid.NullUUID

	// A SessionId is a permanent session identifier, valid for the lifetime of the session.
	SessionId uuid.UUID

	// A PlayerId identifies a particular player in a session.
	//
	// It is generated randomly when a client joins and is only valid in the context of the session.
	// Unlike their [ClientId], the client's PlayerId is shared among other session players.
	PlayerId uuid.UUID

	// A ClientId identifies a particular client.
	// The ClientId is a secret value that is used for access control;
	// for that reason it's never shared with other clients.
	//
	// When a client joins a session, they are assigned a [PlayerId], which, unlike the ClientId, is public.
	ClientId uuid.UUID
)

func (id ImageId) String() string {
	if id.Valid {
		return id.UUID.String()
	} else {
		return "<null>"
	}
}

func (sid SessionId) UUID() uuid.UUID {
	return uuid.UUID(sid)
}

func (id PlayerId) UUID() uuid.UUID {
	return uuid.UUID(id)
}

func (id ClientId) UUID() uuid.UUID {
	return uuid.UUID(id)
}

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

const (
	// InviteCodeAlphabetSize is the cardinality of the alphabet [A-Z0-9].
	InviteCodeAlphabetSize = 36

	InviteCodeLength = 6

	// MaxInviteCodeCount is the total number of possible distinct invite codes.
	// The value is computed as pow(InviteCodeAlphabetSize, InviteCodeLength)
	MaxInviteCodeCount = 2_176_782_336
)

func NewInviteCode() InviteCode {
	var builder strings.Builder
	maxN := big.NewInt(36)

	for i := 0; i < InviteCodeLength; i++ {
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
	state         State
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
	Tx       TxChan
}

type PollOption struct {
	Value TaskAnswer

	// Beneficiaries is a set of players who have submitted this answer and would benefit from having this option win.
	Beneficiaries map[PlayerId]struct{}
}

// An OptionIdx is a "nullable" index for an option in a poll.
// An invalid `OptionIdx` (its zero value) indicates a player has not selected an option yet.
type OptionIdx int

// NewOptionIdx returns a new `OptionIdx`.
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