package account

import "errors"

// The codes of this package's errors, for programs to check with errors.Is.
var (
	ErrRefused = errors.New("account.refused")      // wrong email or password
	ErrServer  = errors.New("account.server-error") // the server failed, or answered what this package can't read
)
