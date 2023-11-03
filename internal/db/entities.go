package db

import "time"

// ImageEntity represents the metadata for images.
// Table - images.
type ImageEntity struct {
	ID string // TODO: find uuid

	// Uploaded default value is false.
	// If image is uploaded the value true.
	Uploaded bool

	// ReadOnly marks that image is read only.
	// If ReadOnly == true upload should fail.
	ReadOnly bool

	OwnerID string // TODO: uuid

	// CreatedAt date and time created.
	CreatedAt time.Time
}

// SessionImageRefsEntity tracks images referenced in live sessions.
// Table - session_image_refs.
type SessionImageRefsEntity struct {
	ImageID   string // TODO: uuid
	SessionID string // TODO: uuid
}

// UserEntity - info about user roles
// Table - users
type UserEntity struct {
	ID string // TODO: uuid

	Role string
}
