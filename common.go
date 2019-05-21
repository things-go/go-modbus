package modbus

import (
	"log"
	"sync/atomic"
)

// 内部调试实现
type logs struct {
	Logger *log.Logger
	// has log output
	haslog uint32
}

// LogMode set enable or diable log output when you has set logger
func (this *logs) LogMode(enable bool) {
	if enable {
		atomic.StoreUint32(&this.haslog, 1)
	} else {
		atomic.StoreUint32(&this.haslog, 0)
	}
}

// logf 格式化输出
func (this *logs) logf(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Printf(format, v...)
	}
}
