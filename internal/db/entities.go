package db

import (
	"github.com/google/uuid"
	"time"
)

// ImageEntity represents the metadata for images.
// Table - images.
type ImageEntity struct {
	ID uuid.NullUUID `db:"id"`

	// Uploaded default value is false.
	// If image is uploaded the value true.
	Uploaded bool `db:"uploaded"`

	// ReadOnly marks that image is read only.
	// If ReadOnly == true upload should fail.
	ReadOnly bool `db:"read_only"`

	OwnerID uuid.NullUUID `db:"owner_id"`

	// CreatedAt date and time created.
	CreatedAt time.Time `db:"created_at"`
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
	ID uuid.NullUUID `db:"id"`

	Role UserRole `db:"role"`
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
	ID uuid.NullUUID `db:"id"`

	OwnerID uuid.NullUUID `db:"owner_id"`
	ImageID uuid.NullUUID `db:"image_id"`

	Name                string `db:"name"`
	Description         string `db:"description"`
	DurationSeconds     int    `db:"duration_secs"`
	PollDurationSeconds int    `db:"poll_duration_secs"`

	PollDurationType PollDurationType `db:"poll_duration_type"`

	TaskKind TaskKind `db:"task_kind"`
}

// CheckedTextTaskEntity - task with TaskKind == CheckedText.
// Relationship 1:1
// Relative table - checked_text_tasks
type CheckedTextTaskEntity struct {
	TaskID uuid.NullUUID `db:"task_id"`

	Answer string `db:"answer"`
}

// ChoiceTaskOptionsEntity - options for task with TaskKind == Choice.
// One choice task can have many options.
// Table - choice_task_options
type ChoiceTaskOptionsEntity struct {
	ID int `db:"id"`

	TaskID uuid.NullUUID `db:"task_id"`

	// Alternative is one of the option for the Task with id == TaskID
	Alternative string `db:"alternative"`

	// Correct shows if this option is true or not
	Correct bool `db:"correct"`
}

// GameEntity represents the game
// Table - games
type GameEntity struct {
	ID uuid.NullUUID `db:"id"`

	Name        string `db:"name"`
	Description string `db:"description"`

	OwnerID uuid.NullUUID `db:"owner_id"`
	ImageID uuid.NullUUID `db:"image_id"` // may be nil

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// GameTaskEntity represents the one task of the game.
// Game can have many tasks, and task can be in many games.
// Table - game_tasks
type GameTaskEntity struct {
	GameID uuid.NullUUID `db:"game_id"`
	TaskID uuid.NullUUID `db:"task_id"`

	// TaskIndex is used to define tasks order in game
	TaskIndex int `db:"task_idx"`
}
