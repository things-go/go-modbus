package modbus

import (
	"errors"
)

// ErrClosedConnection connection has closed.
var ErrClosedConnection = errors.New("use of closed connection")
