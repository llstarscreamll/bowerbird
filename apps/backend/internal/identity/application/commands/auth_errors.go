package commands

import "errors"

var (
	ErrLocalAuthDisabled  = errors.New("local auth is disabled in this environment")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email is not verified")
	ErrInvalidEmail       = errors.New("invalid email")
)
