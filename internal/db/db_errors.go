package db

import "errors"

// RecordNotFound should be used then entity with id is not found
type RecordNotFound struct{}

func (r RecordNotFound) Error() string {
	return "record-not-found"
}

var (
	ErrInvalidDBValue       = errors.New("invalid-db-value")
	ErrToManyEntitiesWithID = errors.New("too-many-entities-with-id")
)

var (
	ErrGeneratedUUIDInvalid = errors.New("generated-uuid-invalid")
	ErrImageIsReadOnly      = errors.New("img-read-only")
	ErrImageIsNotUploaded   = errors.New("img-not-uploaded")
)
