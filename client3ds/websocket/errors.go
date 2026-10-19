package websocket

import "errors"

// The codes of this package's errors, for programs to check with errors.Is.
var (
	ErrHandshake   = errors.New("websocket.handshake-refused")
	ErrNotOpen     = errors.New("websocket.not-open")
	ErrBadFrame    = errors.New("websocket.bad-frame")
	ErrClosed      = errors.New("websocket.closed")
	ErrTooLarge    = errors.New("websocket.message-too-large")
	ErrUnsupported = errors.New("websocket.unsupported-message")
	ErrBadHeader   = errors.New("websocket.bad-header")
)
