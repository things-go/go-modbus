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
func (this *TCPServer) SetReadTimeout(t time.Duration) {
	this.readTimeout = t
}

// SetWriteTimeout set write timeout
func (this *TCPServer) SetWriteTimeout(t time.Duration) {
	this.writeTimeout = t
}

// Close close the server
func (this *TCPServer) Close() error {
	this.mu.Lock()
	if this.listen != nil {
		this.listen.Close()
		this.cancel()
		this.listen = nil
	}
	this.mu.Unlock()
	this.wg.Wait()
	return nil
}

// ListenAndServe 服务
func (this *TCPServer) ListenAndServe(addr string) error {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	this.mu.Lock()
	this.listen = listen
	this.cancel = cancel
	this.mu.Unlock()

	defer func() {
		this.Close()
		this.Error("server stop")
	}()
	this.Debug("server running")
	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		this.wg.Add(1)
		go func() {
			sess := &ServerSession{
				conn,
				this.readTimeout,
				this.writeTimeout,
				this.serverCommon,
				&this.clogs,
			}
			sess.running(ctx)
			this.wg.Done()
		}()
	}
}
