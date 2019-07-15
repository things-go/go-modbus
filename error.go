package modbus

import (
	"errors"
)

// 连接已关闭
var ErrClosedConnection = errors.New("use of closed connection")
