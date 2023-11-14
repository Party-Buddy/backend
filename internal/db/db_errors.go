package db

import "errors"

// RecordNotFound should be used then entity with id is not found
type RecordNotFound struct{}

func (r RecordNotFound) Error() string {
	return "record-not-found"
}

var (
	ErrGeneratedUUIDInvalid = errors.New("generated-uuid-invalid")
)
