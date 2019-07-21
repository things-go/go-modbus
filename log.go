package modbus

import (
	"os"
	"sync/atomic"

	"log"
)

// 内部调试实现
type clogs struct {
	logger LogProvider
	// is log output enabled,1: enable, 0: disable
	hasLog uint32
}

func NewClog() *clogs {
	return NewClogWithPrefix("")
}

func NewClogWithPrefix(prefix string) *clogs {
	return &clogs{
		logger: newDefaultLogger(prefix),
	}
}

// LogMode set enable or disable log output when you has set logger
func (this *clogs) LogMode(enable bool) {
	if enable {
		atomic.StoreUint32(&this.hasLog, 1)
	} else {
		atomic.StoreUint32(&this.hasLog, 0)
	}
}

// SetLogProvider set logger provider
func (this *clogs) SetLogProvider(p LogProvider) {
	if p != nil {
		this.logger = p
	}
}

// Error Log ERROR level message.
func (this *clogs) Error(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.hasLog) == 1 {
		this.logger.Error(format, v...)
	}
}

// Debug Log DEBUG level message.
func (this *clogs) Debug(format string, v ...interface{}) {
	if atomic.LoadUint32(&this.hasLog) == 1 {
		this.logger.Debug(format, v...)
	}
}

// default log
type logger struct {
	*log.Logger
}

var _ LogProvider = (*logger)(nil)

func newDefaultLogger(prefix string) *logger {
	return &logger{
		log.New(os.Stderr, prefix, log.LstdFlags),
	}
}

// Error Log ERROR level message.
func (this *logger) Error(format string, v ...interface{}) {
	this.Printf("[E]: "+format, v...)
}

// Debug Log DEBUG level message.
func (this *logger) Debug(format string, v ...interface{}) {
	this.Printf("[D]: "+format, v...)
}
