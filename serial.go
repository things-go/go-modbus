package modbus

import (
	"io"
	"sync"
	"time"

	"github.com/goburrow/serial"
)

const (
	// SerialDefaultTimeout Serial Default timeout
	SerialDefaultTimeout = 1 * time.Second
	// SerialDefaultAutoReconnect Serial Default auto reconnect count
	SerialDefaultAutoReconnect = 0
)

// serialPort has configuration and I/O controller.
type serialPort struct {
	// Serial port configuration.
	serial.Config
	mu   sync.Mutex
	port io.ReadWriteCloser
	// if > 0, when disconnect,it will try to reconnect the remote
	// but if we active close self,it will not to reconncet
	// if == 0 auto reconnect not active
	autoReconnect byte
}

// Connect try to connect the remote server
func (this *serialPort) Connect() error {
	this.mu.Lock()
	err := this.connect()
	this.mu.Unlock()
	return err
}

// Caller must hold the mutex before calling this method.
func (this *serialPort) connect() error {
	port, err := serial.Open(&this.Config)
	if err != nil {
		return err
	}
	this.port = port
	return nil
}

// IsConnected returns a bool signifying whether the client is connected or not.
func (this *serialPort) IsConnected() bool {
	this.mu.Lock()
	b := this.isConnected()
	this.mu.Unlock()
	return b
}

// Caller must hold the mutex before calling this method.
func (this *serialPort) isConnected() bool {
	return this.port != nil
}

// SetAutoReconnect set auto reconnect count
// if cnt == 0, disable auto reconnect
// if cnt > 0 ,enable auto reconnect,but max 6
func (this *serialPort) SetAutoReconnect(cnt byte) {
	this.mu.Lock()
	this.autoReconnect = cnt
	if this.autoReconnect > 6 {
		this.autoReconnect = 6
	}
	this.mu.Unlock()
}

// Close close current connection.
func (this *serialPort) Close() error {
	var err error
	this.mu.Lock()
	if this.port != nil {
		err = this.port.Close()
		this.port = nil
	}
	this.mu.Unlock()
	return err
}
