package schemas

import (
	"github.com/google/uuid"
	"time"
)

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

	PollDuration PollDuration `json:"poll-duration,omitempty"`
}

type BaseTaskWithImg struct {
	BaseTask
	ImgURI string `json:"img-uri,omitempty"`
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

var validTaskTypes = []TaskType{Photo, Text, CheckedText, Choice}

const (
	Photo       TaskType = "photo"
	Text        TaskType = "text"
	CheckedText TaskType = "checked-text"
	Choice      TaskType = "choice"
)

type GameDetails struct {
	BaseGameInfo

	Tasks []BaseTaskWithImg `json:"tasks"`
}

type BaseTaskWithImgAndID struct {
	BaseTaskWithImg

	ID          uuid.UUID `json:"id"`
	LastUpdated time.Time `json:"last-updated"`
}

type IDGameInfo struct {
	BaseGameInfo

	ID uuid.UUID `json:"id"`

	Tasks []BaseTaskWithImgAndID `json:"tasks"`
}
