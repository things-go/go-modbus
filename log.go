package modbus

import (
	"github.com/thinkgos/library/elog"
)

// 内部调试实现
type logs struct {
	*elog.Elog
}

// LogMode set enable or diable log output when you has set logger
func (this *logs) LogMode(enable bool) {
	this.Mode(enable)
}

// SetLogProvider set logger provider
func (this *logs) SetLogProvider(p elog.Provider) {
	this.SetProvider(p)
}

// SetLogger set logger
func (this *logs) SetLogger(l *elog.Elog) {
	if l != nil {
		this.Elog = l
	}
}
