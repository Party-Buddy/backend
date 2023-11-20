package session

import "time"

type State interface {
	isState() // an unexported marker method so we don't have scary interface{}s floating around
}

// An AwaitingPlayersState is an initial session state during which the game is not yet started.
// New players can discover the session via its invite code or session id, only the latter of which is permanent.
// (The invite code expires once the game starts — or the session is closed before it starts if the owner quits.)
type AwaitingPlayersState struct {
	// A short code used for session discovery.
	InviteCode InviteCode

	// A set of players who expressed their readiness.
	PlayersReady map[PlayerId]struct{}

	// Whether all players need to be ready before the game can start.
	RequireReady bool

	// The creator of the session.
	// While waiting for players, they have additional privileges: for exmaple, they can remove people from the session.
	// Of course, with great power comes great responsibility: if this player leaves, the session will be closed.
	Owner PlayerId
}

func (*AwaitingPlayersState) isState() {}

// A GameStartedState is a state right after the game starts.
// No task is currently underway: we're just giving people time to react and mentally prepare.
type GameStartedState struct {
	// When the first start should start.
	Deadline time.Time
}

func (*GameStartedState) isState() {}

// A TaskStartedState corresponds to a session state while a game task is in progress.
// Players are able to update their answers, possibly marking them ready as well.
type TaskStartedState struct {
	// The index of the current task.
	TaskIdx int

	// When the task ends.
	Deadline time.Time

	// The players' current answers.
	Answers map[PlayerId]TaskAnswer

	// A set of players that expressed their readiness.
	Ready map[PlayerId]struct{}
}

func (*TaskStartedState) isState() {}

// A PollStartedState is a state while players vote for each other's answers.
// Some tasks do not call for a poll — in that case this state is simply skipped.
type PollStartedState struct {
	// The index of the current task.
	TaskIdx int

	// When the poll ends.
	Deadline time.Time

	// The options to choose from.
	Options []PollOption

	// Which options (represented by their indices into `options`) people chose.
	Votes map[PlayerId]OptionIdx
}

func (*PollStartedState) isState() {}

// A TaskEndedState is a state right after a task ends.
// This gives players time to reflect on their performance and envy their peers.
type TaskEndedState struct {
	// The index of the ended task.
	TaskIdx int

	// When the next task starts (or, if the ended task was the last one, the game ends).
	Deadline time.Time

	// The answers made by players — and the popularity of those answers.
	Results []AnswerResult
}

func (*TaskEndedState) isState() {}

// "But," you may ask, "what about the game-ended state?"
// We remove a session once it reaches this state, so representing it is unnecessary.
// Thus we don't.

// Assert all of these are states (to catch missing methods).
var (
	_ State = &AwaitingPlayersState{}
	_ State = &GameStartedState{}
	_ State = &TaskStartedState{}
	_ State = &PollStartedState{}
	_ State = &TaskEndedState{}
)
