package storage

import "time"

type state interface {
	isState() // an unexported marker method so we don't have scary interface{}s floating around
}

// An awaitingPlayersState is an initial session state during which the game is not yet started.
// New players can discover the session via its invite code or session id, only the latter of which is permanent.
// (The invite code expires once the game starts — or the session is closed before it starts if the owner quits.)
type awaitingPlayersState struct {
	// A short code used for session discovery.
	inviteCode InviteCode

	// A set of players who expressed their readiness.
	playersReady map[PlayerId]struct{}

	// Whether all players need to be ready before the game can start.
	requireReady bool

	// The creator of the session.
	// While waiting for players, they have additional privileges: for exmaple, they can remove people from the session.
	// Of course, with great power comes great responsibility: if this player leaves, the session will be closed.
	owner PlayerId
}

func (*awaitingPlayersState) isState() {}

// A gameStartedState is a state right after the game starts.
// No task is currently underway: we're just giving people time to react and mentally prepare.
type gameStartedState struct {
	// When the first start should start.
	deadline time.Time
}

func (*gameStartedState) isState() {}

// A taskStartedState corresponds to a session state while a game task is in progress.
// Players are able to update their answers, possibly marking them ready as well.
type taskStartedState struct {
	// The index of the current task.
	taskIdx int

	// When the task ends.
	deadline time.Duration

	// The players' current answers.
	answers map[PlayerId]TaskAnswer

	// A set of players that expressed their readiness.
	ready map[PlayerId]struct{}
}

func (*taskStartedState) isState() {}

// A pollStartedState is a state while players vote for each other's answers.
// Some tasks do not call for a poll — in that case this state is simply skipped.
type pollStartedState struct {
	// The index of the current task.
	taskIdx int

	// When the poll ends.
	deadline time.Duration

	// The options to choose from.
	options []PollOption

	// Which options (represented by their indices into `options`) people chose.
	votes map[PlayerId]OptionIdx
}

func (*pollStartedState) isState() {}

// A taskEndedState is a state right after a task ends.
// This gives players time to reflect on their performance and envy their peers.
type taskEndedState struct {
	// The index of the ended task.
	taskIdx int

	// When the next task starts (or, if the ended task was the last one, the game ends).
	deadline time.Duration

	// The answers made by players — and the popularity of those answers.
	results []AnswerResult
}

func (*taskEndedState) isState() {}

// "But," you may ask, "what about the game-ended state?"
// We remove a session once it reaches this state, so representing it is unnecessary.
// Thus we don't.

// Assert all of these are states (to catch missing methods).
var (
	_ state = &awaitingPlayersState{}
	_ state = &gameStartedState{}
	_ state = &taskStartedState{}
	_ state = &pollStartedState{}
	_ state = &taskEndedState{}
)
