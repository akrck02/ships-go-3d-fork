package web

import "errors"

// The codes of this package's errors, for programs to check with errors.Is.
var (
	ErrTimeout     = errors.New("web.timeout")
	ErrBadResponse = errors.New("web.bad-response")
	ErrBadHeader   = errors.New("web.bad-header")
)
