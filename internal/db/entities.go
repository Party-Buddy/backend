package db

import (
	"github.com/google/uuid"
	"time"
)

// ImageEntity represents the metadata for images.
// Table - images.
type ImageEntity struct {
	ID uuid.NullUUID

	// Uploaded default value is false.
	// If image is uploaded the value true.
	Uploaded bool

	// ReadOnly marks that image is read only.
	// If ReadOnly == true upload should fail.
	ReadOnly bool

	OwnerID uuid.NullUUID

	// CreatedAt date and time created.
	CreatedAt time.Time
}

// SessionImageRefsEntity tracks images referenced in live sessions.
// Table - session_image_refs.
type SessionImageRefsEntity struct {
	ImageID   uuid.NullUUID
	SessionID uuid.NullUUID
}

// ImageRefsEntity - is used for tracking image using.
// View - image_refs_view
type ImageRefsEntity struct {
	ImageID uuid.NullUUID

	// RefCount is the number of public games + sessions + tasks
	// which use the image with id == ImageID
	RefCount int
}

type UserRole string

const (
	Admin UserRole = "admin"
	Base  UserRole = "base"
)

// UserEntity - info about user roles
// Table - users
type UserEntity struct {
	ID uuid.NullUUID

	Role string
}

type TaskKind string

const (
	Photo       TaskKind = "photo"
	Text        TaskKind = "text"
	CheckedText TaskKind = "checked-text"
	Choice      TaskKind = "choice"
)

type PollDurationType string

const (
	Fixed   PollDurationType = "fixed"
	Dynamic PollDurationType = "dynamic"
)

// TaskEntity - provides info about task
// Table - tasks
type TaskEntity struct {
	ID uuid.NullUUID

	OwnerID uuid.NullUUID
	ImageID uuid.NullUUID

	Name                string
	Description         string
	DurationSeconds     int
	PollDurationSeconds int

	PollDurationType PollDurationType

	TaskKind TaskKind
}

// CheckedTextTaskEntity - task with TaskKind == CheckedText.
// Relationship 1:1
// Relative table - checked_text_tasks
type CheckedTextTaskEntity struct {
	TaskEntity // field TaskKind should be CheckedText

	Answer string
}

// ChoiceTaskOptionsEntity - options for task with TaskKind == Choice.
// One choice task can have many options.
// Table - choice_task_options
type ChoiceTaskOptionsEntity struct {
	TaskID uuid.NullUUID

	// Alternative is one of the option for the Task with id == TaskID
	Alternative string

	// Correct shows if this option is true or not
	Correct bool
}

// GameEntity represents the game
// Table - games
type GameEntity struct {
	ID uuid.NullUUID

	Name        string
	Description string

	OwnerID uuid.NullUUID
	ImageID uuid.NullUUID // may be nil

	CreatedAt time.Time
	UpdatedAt time.Time
}
