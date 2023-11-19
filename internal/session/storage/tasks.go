package storage

import "time"

// Tasks

type Task interface {
	Name() string
	Description() string
	ImageId() ImageId
	isTask() // an unexported marker method to indicate a type is in fact a task
}

type BaseTask struct {
	Name        string
	Description string
	ImageId     ImageId
	TaskDuration time.Duration
}

type PollTask struct {
	Task

	PollDuration PollDurationer
}

type PhotoTask PollTask
type TextTask PollTask

type CheckedTextTask struct {
	Task

	Answer string
}

type ChoiceTask struct {
	Task

	Options   []string
	AnswerIdx int
}

func (PhotoTask) isTask()       {}
func (TextTask) isTask()        {}
func (CheckedTextTask) isTask() {}
func (ChoiceTask) isTask()      {}

// type assertions
var (
	_ Task = PhotoTask{}
	_ Task = TextTask{}
	_ Task = CheckedTextTask{}
	_ Task = ChoiceTask{}
)

// Task answers

type TaskAnswer interface {
	isTaskAnswer() // an unexported marker method so we can see what types really are task answers
}

type (
	PhotoTaskAnswer   ImageId
	TextTaskAnswer    string
	CheckedTextAnswer string
	ChoiceTaskAnswer  int
)

func (PhotoTaskAnswer) isTaskAnswer()   {}
func (TextTaskAnswer) isTaskAnswer()    {}
func (CheckedTextAnswer) isTaskAnswer() {}
func (ChoiceTaskAnswer) isTaskAnswer()  {}

// Poll durations

type PollDurationer interface {
	// PollDuration calculates a poll duration for the given session.
	// The manager mutex must be locked before calling this method.
	PollDuration(s *UnsafeStorage, sid SessionId) time.Duration
}

// A FixedPollDuration computes a duration that does not depend on the session state.
type FixedPollDuration time.Duration

func (d FixedPollDuration) PollDuration(s *UnsafeStorage, sid SessionId) time.Duration {
	return time.Duration(d)
}

// A DynamicPollDuration computes a duration which dynamically scales depending on how many players are in the game.
// The total duration is calculated as `playerCount * timePerPlayer`.
type DynamicPollDuration time.Duration

func (d DynamicPollDuration) PollDuration(s *UnsafeStorage, sid SessionId) time.Duration {
	return time.Duration(d) * time.Duration(s.PlayerCount(sid))
}
