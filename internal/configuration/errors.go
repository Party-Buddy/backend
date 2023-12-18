package configuration

import "errors"

var (
	ErrDBUserInfoNotProvided = errors.New("db-user-info-not-provided")
	ErrDBHostNotProvided     = errors.New("db-host-not-provided")
	ErrDBPortNotProvided     = errors.New("db-port-not-provided")
	ErrDBNameNotProvided     = errors.New("db-name-not-provided")
)
