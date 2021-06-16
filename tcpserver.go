package modbus

import (
	"context"
	"net"
	"sync"
	"time"
)

// TCP Default read & write timeout
const (
	TCPDefaultReadTimeout  = 60 * time.Second
	TCPDefaultWriteTimeout = 1 * time.Second
)

// TCPServer modbus tcp server
type TCPServer struct {
	mu           sync.Mutex
	listen       net.Listener
	wg           sync.WaitGroup
	cancel       context.CancelFunc
	readTimeout  time.Duration
	writeTimeout time.Duration
	*serverCommon
	logger
}

// NewTCPServer the modbus server listening on "address:port".
func NewTCPServer() *TCPServer {
	return &TCPServer{
		readTimeout:  TCPDefaultReadTimeout,
		writeTimeout: TCPDefaultWriteTimeout,
		serverCommon: newServerCommon(),
		logger:       newLogger("modbusTCPServer => "),
	}
}

// SetReadTimeout set read timeout
func (sf *TCPServer) SetReadTimeout(t time.Duration) *TCPServer {
	sf.readTimeout = t
	return sf
}

// SetWriteTimeout set write timeout
func (sf *TCPServer) SetWriteTimeout(t time.Duration) *TCPServer {
	sf.writeTimeout = t
	return sf
}

// Close close the server until all server close then return
func (sf *TCPServer) Close() error {
	sf.mu.Lock()
	if sf.listen != nil {
		sf.listen.Close()
		sf.cancel()
		sf.listen = nil
	}
	sf.mu.Unlock()
	sf.wg.Wait()
	return nil
}

const minTempDelay = 5 * time.Millisecond

// ListenAndServe listen and server
func (sf *TCPServer) ListenAndServe(addr string) error {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	sf.mu.Lock()
	sf.listen = listen
	sf.cancel = cancel
	sf.mu.Unlock()

	sf.Debugf("server started,and listen address: %s", addr)
	defer func() {
		sf.Close()
		sf.Debugf("server stopped")
	}()
	var tempDelay = minTempDelay // how long to sleep on accept failure

	for {
		conn, err := listen.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				tempDelay <<= 1
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = minTempDelay
		sf.wg.Add(1)
		go func() {
			sess := &ServerSession{
				conn,
				sf.readTimeout,
				sf.writeTimeout,
				sf.serverCommon,
				sf.logger,
			}
			sess.running(ctx)
			sf.wg.Done()
		}()
	}
}
