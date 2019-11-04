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

// newClogWithPrefix new clog with prefix
func newClogWithPrefix(prefix string) *clogs {
	return &clogs{
		logger: newDefaultLogger(prefix),
	}
}

// LogMode set enable or disable log output when you has set logger
func (sf *clogs) LogMode(enable bool) {
	if enable {
		atomic.StoreUint32(&sf.hasLog, 1)
	} else {
		atomic.StoreUint32(&sf.hasLog, 0)
	}
}

// SetLogProvider set logger provider
func (sf *clogs) SetLogProvider(p LogProvider) {
	if p != nil {
		sf.logger = p
	}
}

// Error Log ERROR level message.
func (sf *clogs) Error(format string, v ...interface{}) {
	if atomic.LoadUint32(&sf.hasLog) == 1 {
		sf.logger.Error(format, v...)
	}
}

// Debug Log DEBUG level message.
func (sf *clogs) Debug(format string, v ...interface{}) {
	if atomic.LoadUint32(&sf.hasLog) == 1 {
		sf.logger.Debug(format, v...)
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
func (sf *logger) Error(format string, v ...interface{}) {
	sf.Printf("[E]: "+format, v...)
}

// Debug Log DEBUG level message.
func (sf *logger) Debug(format string, v ...interface{}) {
	sf.Printf("[D]: "+format, v...)
}
