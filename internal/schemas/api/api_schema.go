package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"party-buddy/internal/validate"

	"github.com/cohesivestack/valgo"
)

type ErrorKind string

func (e ErrorKind) MarshalText() ([]byte, error) {
	return []byte(e), nil
}

// AuthorizationErrorKind codes
var (
	ErrUserIdInvalid       ErrorKind = "user-id-invalid"
	ErrOnlyOwnerAllowed    ErrorKind = "only-owned-allowed"
	ErrAuthRequired        ErrorKind = "auth-required"
	ErrNotEnoughPrivileges ErrorKind = "not-enough-privileges"
)

// ImageErrorKind codes
var (
	ErrImgNotProvided       ErrorKind = "img-not-provided"
	ErrImgTooLarge          ErrorKind = "img-too-large"
	ErrImgFormatUnsupported ErrorKind = "img-format-unsupported"
	ErrImgMalformed         ErrorKind = "img-malformed"
	ErrImgUploadForbidden   ErrorKind = "img-upload-forbidden"
)

// InvalidParamErrorKind codes
var (
	ErrSchemaInvalid ErrorKind = "schema-invalid"
	ErrParamMissing  ErrorKind = "param-missing"
	ErrParamInvalid  ErrorKind = "param-invalid"
)

// TaskErrorKind codes
var (
	ErrTaskNotFound ErrorKind = "task-not-found"
	ErrTaskInvalid  ErrorKind = "task-invalid"
	ErrTaskUsed     ErrorKind = "task-used"
)

// GameErrorKind codes
var (
	ErrGameInvalid ErrorKind = "game-invalid"
)

// GeneralErrorKind codes
var (
	ErrNotFound         ErrorKind = "not-found"
	ErrMethodNotAllowed ErrorKind = "method-not-allowed"
	ErrInternal         ErrorKind = "internal"
	ErrMalformedRequest ErrorKind = "malformed-request"
)

var (
	ErrInvalidUpgrade ErrorKind = "invalid-upgrade"
	ErrUpgradeFailed  ErrorKind = "upgrade-failed"
)

type Error struct {
	Kind    ErrorKind `json:"error"`
	Message string    `json:"message"`
}

func Errorf(kind ErrorKind, format string, a ...any) *Error {
	return &Error{
		Kind:    kind,
		Message: fmt.Sprintf(format, a...),
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v (code `%v`)", e.Message, e.Kind)
}

func (e *Error) String() string {
	return e.Message
}

// How to add a new response (output, server-to-client) body struct:
// 1. Create a new (exported) type of the desired structure.
//    All fields should be exported.
// 2. Set a JSON tag for each field.
// 3. Use [json.Marshal] (or its encoder) to convert a structure to JSON.
//
// How to add a new request (input, client-to-server) body struct:
// 1. Create a new (exported) type of the desired structure.
//    All fields should be exported for consistency.
//    Use pointer types for fields (e.g., `*string`, not `string`).
// 2. Set a JSON tag for each field.
// 3. Implement [validate.Validator].
//    Make sure to check each required field for nil ([validate.FieldValue] may come in handy for this):
//    otherwise you won't know if this field was present in a request.
// 4. Use [Parse] to deserialize JSON into a target structure.

// Parse parses JSON-encoded data into target and runs validation.
// Returns a formatted [Error] on parsing failure.
func Parse(ctx context.Context, target validate.Validator, data []byte, allowUnknownFields bool) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if !allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &typeError) {
			return Errorf(
				ErrMalformedRequest,
				"in field `%s`: %s has an illegal type",
				typeError.Field,
				typeError.Value,
			)
		}

		return Errorf(ErrMalformedRequest, "request body is not valid JSON: %s", err)
	}
	if decoder.InputOffset() < int64(len(data)) {
		return Errorf(ErrMalformedRequest, "request body is not valid JSON: trailing junk")
	}

	if val := target.Validate(ctx); !val.Valid() {
		if fieldName, msg, ok := validate.ExtractValgoErrorFields(val.Error().(*valgo.Error)); ok {
			return Errorf(ErrMalformedRequest, "in field `%s`: %s", fieldName, msg)
		} else {
			return Errorf(ErrMalformedRequest, "malformed request body")
		}
	}

	return nil
}
