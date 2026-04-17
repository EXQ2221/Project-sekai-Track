package errcode

import "errors"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrNotFound     = errors.New("not found")
	ErrInternal     = errors.New("internal server error")

	ErrUsernameIncorrect = errors.New("username incorrect")
	ErrPasswordIncorrect = errors.New("password incorrect")
	ErrSessionRevoked    = errors.New("session revoked")
	ErrRefreshReuse      = errors.New("refresh token reuse detected")
	ErrDeviceMismatch    = errors.New("device or browser mismatch")
)
