package modbus

import (
	"io"
	"sync"
	"time"

	"github.com/goburrow/serial"
)

// SerialDefaultTimeout Serial Default timeout
const SerialDefaultTimeout = 1 * time.Second

// serialPort has configuration and I/O controller.
type serialPort struct {
	// Serial port configuration.
	serial.Config
	mu   sync.Mutex
	port io.ReadWriteCloser
}

// Connect try to connect the remote server
func (sf *serialPort) Connect() (err error) {
	sf.mu.Lock()
	err = sf.connect()
	sf.mu.Unlock()
	return
}

// Caller must hold the mutex before calling this method.
func (sf *serialPort) connect() error {
	if sf.port == nil {
		port, err := serial.Open(&sf.Config)
		if err != nil {
			return err
		}
		sf.port = port
	}
	return nil
}

// IsConnected returns a bool signifying whether the client is connected or not.
func (sf *serialPort) IsConnected() (b bool) {
	sf.mu.Lock()
	b = sf.port != nil
	sf.mu.Unlock()
	return b
}

// setSerialConfig set serial config
func (sf *serialPort) setSerialConfig(config serial.Config) {
	sf.Config = config
}

func (sf *serialPort) setTCPTimeout(time.Duration) {}

func (sf *serialPort) close() (err error) {
	if sf.port != nil {
		err = sf.port.Close()
		sf.port = nil
	}
	return err
}

// Close close current connection.
func (sf *serialPort) Close() (err error) {
	sf.mu.Lock()
	err = sf.close()
	sf.mu.Unlock()
	return
}
