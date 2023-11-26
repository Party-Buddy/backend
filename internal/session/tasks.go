package session

import "time"

// Tasks

type Task interface {
	GetName() string
	GetDescription() string
	GetImageID() ImageID
	GetTaskDuration() time.Duration
	isTask() // an unexported marker method to indicate a type is in fact a task
}

type BaseTask struct {
	Name         string
	Description  string
	ImageID      ImageID
	TaskDuration time.Duration
}

type PollTask struct {
	BaseTask

	PollDuration PollDurationer
}

type PhotoTask PollTask

func (t PhotoTask) GetImageID() ImageID {
	return t.ImageID
}

func (t PhotoTask) GetName() string {
	return t.Name
}

func (t PhotoTask) GetDescription() string {
	return t.Description
}

func (t PhotoTask) GetTaskDuration() time.Duration {
	return t.TaskDuration
}

type TextTask PollTask

func (t TextTask) GetImageID() ImageID {
	return t.ImageID
}

func (t TextTask) GetName() string {
	return t.Name
}

func (t TextTask) GetDescription() string {
	return t.Description
}

func (t TextTask) GetTaskDuration() time.Duration {
	return t.TaskDuration
}

type CheckedTextTask struct {
	BaseTask

	Answer string
}

func (t CheckedTextTask) GetImageID() ImageID {
	return t.ImageID
}

func (t CheckedTextTask) GetName() string {
	return t.Name
}

func (t CheckedTextTask) GetDescription() string {
	return t.Description
}

func (t CheckedTextTask) GetTaskDuration() time.Duration {
	return t.TaskDuration
}

type ChoiceTask struct {
	BaseTask

	Options   []string
	AnswerIdx int
}

func (t ChoiceTask) GetImageID() ImageID {
	return t.ImageID
}

func (t ChoiceTask) GetName() string {
	return t.Name
}

func (t ChoiceTask) GetDescription() string {
	return t.Description
}

func (t ChoiceTask) GetTaskDuration() time.Duration {
	return t.TaskDuration
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
	PhotoTaskAnswer   ImageID
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
	PollDuration(s *UnsafeStorage, sid SessionID) time.Duration
}

// A FixedPollDuration computes a duration that does not depend on the session state.
type FixedPollDuration time.Duration

func (d FixedPollDuration) PollDuration(s *UnsafeStorage, sid SessionID) time.Duration {
	return time.Duration(d)
}

// A DynamicPollDuration computes a duration which dynamically scales depending on how many players are in the game.
// The total duration is calculated as `playerCount * timePerPlayer`.
type DynamicPollDuration time.Duration

func (d DynamicPollDuration) PollDuration(s *UnsafeStorage, sid SessionID) time.Duration {
	return time.Duration(d) * time.Duration(s.PlayerCount(sid))
}
