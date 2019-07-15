package modbus

import (
	"errors"
)

// ErrClosedConnection 连接已关闭
var ErrClosedConnection = errors.New("use of closed connection")
