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
func (sf *serialPort) Connect() error {
	sf.mu.Lock()
	err := sf.connect()
	sf.mu.Unlock()
	return err
}

// Caller must hold the mutex before calling this method.
func (sf *serialPort) connect() error {
	port, err := serial.Open(&sf.Config)
	if err != nil {
		return err
	}
	sf.port = port
	return nil
}

// IsConnected returns a bool signifying whether the client is connected or not.
func (sf *serialPort) IsConnected() bool {
	sf.mu.Lock()
	b := sf.isConnected()
	sf.mu.Unlock()
	return b
}

// Caller must hold the mutex before calling this method.
func (sf *serialPort) isConnected() bool {
	return sf.port != nil
}

// SetAutoReconnect set auto reconnect count
// if cnt == 0, disable auto reconnect
// if cnt > 0 ,enable auto reconnect,but max 6
func (sf *serialPort) SetAutoReconnect(cnt byte) {
	sf.mu.Lock()
	sf.autoReconnect = cnt
	if sf.autoReconnect > 6 {
		sf.autoReconnect = 6
	}
	sf.mu.Unlock()
}

// Close close current connection.
func (sf *serialPort) Close() error {
	var err error
	sf.mu.Lock()
	if sf.port != nil {
		err = sf.port.Close()
		sf.port = nil
	}
	sf.mu.Unlock()
	return err
}
