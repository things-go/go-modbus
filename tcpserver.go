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
	clogs
}

// NewTCPServer the modbus server listening on "address:port".
func NewTCPServer() *TCPServer {
	return &TCPServer{
		readTimeout:  TCPDefaultReadTimeout,
		writeTimeout: TCPDefaultWriteTimeout,
		serverCommon: newServerCommon(),
		clogs:        clogs{newDefaultLogger("modbusTCPServer =>"), 0},
	}
}

// SetReadTimeout set read timeout
func (sf *TCPServer) SetReadTimeout(t time.Duration) {
	sf.readTimeout = t
}

// SetWriteTimeout set write timeout
func (sf *TCPServer) SetWriteTimeout(t time.Duration) {
	sf.writeTimeout = t
}

// Close close the server
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

// ListenAndServe 服务
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

	defer func() {
		sf.Close()
		sf.Error("server stop")
	}()
	sf.Debug("server running")
	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		sf.wg.Add(1)
		go func() {
			sess := &ServerSession{
				conn,
				sf.readTimeout,
				sf.writeTimeout,
				sf.serverCommon,
				&sf.clogs,
			}
			sess.running(ctx)
			sf.wg.Done()
		}()
	}
}
