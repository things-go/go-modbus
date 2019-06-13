package modbus

import (
	"sync/atomic"
)

// 内部调试实现
type logs struct {
	Logger LogProvider
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

// SetLogProvider set logger provider
func (this *logs) SetLogProvider(p LogProvider) {
	this.Logger = p
}

// Emergency Log EMERGENCY level message.
func (this *logs) Emergency(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Emergency(format, v...)
	}
}

// Alert Log ALERT level message.
func (this *logs) Alert(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Alert(format, v...)
	}
}

// Critical Log CRITICAL level message.
func (this *logs) Critical(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Critical(format, v...)
	}
}

// Error Log ERROR level message.
func (this *logs) Error(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Error(format, v...)
	}
}

// Warning Log WARNING level message.
func (this *logs) Warning(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Warning(format, v...)
	}
}

// Notice Log NOTICE level message.
func (this *logs) Notice(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Notice(format, v...)
	}
}

// Informational Log INFORMATIONAL level message.
func (this *logs) Informational(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Informational(format, v...)
	}
}

// Debug Log DEBUG level message.
func (this *logs) Debug(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.haslog) == 1 && this.Logger != nil {
		this.Logger.Debug(format, v...)
	}
}
