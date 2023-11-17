package api

import "fmt"

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
	ErrNotFound ErrorKind = "not-found"
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