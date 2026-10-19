package protocol

import "errors"

// ErrUnreadable is a message from the server that isn't the JSON this package expects.
var ErrUnreadable = errors.New("protocol.unreadable-message")
