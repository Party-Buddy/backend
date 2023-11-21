package schemas

import "time"

type BaseGameInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ImgURI      string    `json:"img-uri"`
	DateChanged time.Time `json:"date-changed"`
}

type SchemaTask interface {
	isSchemaTask()
}

type BaseTask struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Duration must be Fixed
	Duration DurationType `json:"duration"`

	Type TaskType `json:"type"`

	ImgURI string `json:"img-uri,omitempty"`

	PollDuration DurationType `json:"poll-duration,omitempty"`
}

func (*BaseTask) isSchemaTask() {}

type DurationKindType string

const (
	Fixed   DurationKindType = "fixed"
	Dynamic DurationKindType = "dynamic"
)

type DurationType struct {
	Kind DurationKindType `json:"kind"`
	Secs uint16           `json:"secs"`
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

	Tasks []SchemaTask `json:"tasks"`
}
