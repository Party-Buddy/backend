package schemas

import "time"

type BaseGameInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ImgURI      string    `json:"img-uri"`
	DateChanged time.Time `json:"date-changed"`
}

type BaseTask struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Duration must be Fixed
	Duration PollDuration `json:"duration"`

	Type TaskType `json:"type"`

	ImgURI string `json:"img-uri,omitempty"`

	PollDuration PollDuration `json:"poll-duration,omitempty"`
}

type DurationKind string

const (
	Fixed   DurationKind = "fixed"
	Dynamic DurationKind = "dynamic"
)

type PollDuration struct {
	Kind DurationKind `json:"kind"`
	Secs uint16       `json:"secs"`
}

type TaskType string

const (
	Photo       TaskType = "photo"
	Text        TaskType = "text"
	CheckedText TaskType = "checked-text"
	Choice      TaskType = "choice"
)

type GameDetails struct {
	BaseGameInfo

	Tasks []BaseTask `json:"tasks"`
}
