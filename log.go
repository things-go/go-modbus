package modbus

import (
	"sync/atomic"

	"log"
)

// 内部调试实现
type logs struct {
	logger LogProvider
	// is log output enabled,1: enable, 0: disable
	hasLog uint32
}

// LogMode set enable or disable log output when you has set logger
func (this *logs) LogMode(enable bool) {
	if enable {
		atomic.StoreUint32(&this.hasLog, 1)
	} else {
		atomic.StoreUint32(&this.hasLog, 0)
	}
}

// SetLogProvider set logger provider
func (this *logs) SetLogProvider(p LogProvider) {
	if p != nil {
		this.logger = p
	}
}

// Error Log ERROR level message.
func (this *logs) Error(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.hasLog) == 1 {
		this.logger.Error(format, v...)
	}
}

// Debug Log DEBUG level message.
func (this *logs) Debug(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.hasLog) == 1 {
		this.logger.Debug(format, v...)
	}
}

// default log
type logger struct{}

var _ LogProvider = (*logger)(nil)

func newLogger() *logger {
	return &logger{}
}

// Error Log ERROR level message.
func (this *logger) Error(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Debug Log DEBUG level message.
func (this *logger) Debug(format string, v ...interface{}) {
	log.Printf(format, v...)
}
