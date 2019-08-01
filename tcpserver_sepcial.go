package modbus

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// defined default value
const (
	DefaultConnectTimeout    = 15 * time.Second
	DefaultReconnectInterval = 1 * time.Minute
)
const (
	initial uint32 = iota
	disconnected
	connected
)

// TCPServerSpecial server special interface
type TCPServerSpecial interface {
	UnderlyingConn() net.Conn
	IsConnected() bool
	IsClosed() bool
	Start() error
	io.Closer

	SetTLSConfig(t *tls.Config)
	AddRemoteServer(server string) error
	SetConnectTimeout(t time.Duration)
	SetReconnectInterval(t time.Duration)
	EnableAutoReconnect(b bool)
	SetReadTimeout(t time.Duration)
	SetWriteTimeout(t time.Duration)
	SetOnConnectHandler(f OnConnectHandler)
	SetConnectionLostHandler(f ConnectionLostHandler)

	LogMode(enable bool)
	SetLogProvider(p LogProvider)
}

// OnConnectHandler when connected it will be call
type OnConnectHandler func(c TCPServerSpecial) error

// ConnectionLostHandler when Connection lost it will be call
type ConnectionLostHandler func(c TCPServerSpecial)

// tcpServerSpecial modbus tcp server special
type tcpServerSpecial struct {
	ServerSession
	server            *url.URL // 连接的服务器端
	rwMux             sync.RWMutex
	status            uint32
	connectTimeout    time.Duration // 连接超时时间
	autoReconnect     bool          // 是否启动重连
	reconnectInterval time.Duration // 重连间隔时间
	TLSConfig         *tls.Config
	onConnect         OnConnectHandler
	onConnectionLost  ConnectionLostHandler
	cancel            context.CancelFunc
}

// NewTCPServerSpecial new tcp server special
func NewTCPServerSpecial() *tcpServerSpecial {
	return &tcpServerSpecial{
		ServerSession: ServerSession{
			readTimeout:  TCPDefaultReadTimeout,
			writeTimeout: TCPDefaultWriteTimeout,
			serverCommon: newServerCommon(),
			clogs:        NewClogWithPrefix("modbusTCPServerSpec =>"),
		},
		connectTimeout:    DefaultConnectTimeout,
		autoReconnect:     true,
		reconnectInterval: DefaultReconnectInterval,
		onConnect:         func(TCPServerSpecial) error { return nil },
		onConnectionLost:  func(TCPServerSpecial) {},
	}
}

// UnderlyingConn go underlying tcp conn
func (this *tcpServerSpecial) UnderlyingConn() net.Conn {
	return this.conn
}

// SetConnectTimeout set tcp connect the host timeout
func (this *tcpServerSpecial) SetConnectTimeout(t time.Duration) {
	this.connectTimeout = t
}

// SetReconnectInterval set tcp  reconnect the host interval when connect failed after try
func (this *tcpServerSpecial) SetReconnectInterval(t time.Duration) {
	this.reconnectInterval = t
}

func (this *tcpServerSpecial) EnableAutoReconnect(b bool) {
	this.autoReconnect = b
}

// SetTLSConfig set tls config
func (this *tcpServerSpecial) SetTLSConfig(t *tls.Config) {
	this.TLSConfig = t
}

// SetReadTimeout set read timeout
func (this *ServerSession) SetReadTimeout(t time.Duration) {
	this.readTimeout = t
}

// SetWriteTimeout set write timeout
func (this *ServerSession) SetWriteTimeout(t time.Duration) {
	this.writeTimeout = t
}

// SetOnConnectHandler set on connect handler
func (this *tcpServerSpecial) SetOnConnectHandler(f OnConnectHandler) {
	if f != nil {
		this.onConnect = f
	}
}

// SetConnectionLostHandler set connection lost handler
func (this *tcpServerSpecial) SetConnectionLostHandler(f ConnectionLostHandler) {
	if f != nil {
		this.onConnectionLost = f
	}
}

// AddRemoteServer adds a broker URI to the list of brokers to be used.
// The format should be scheme://host:port
// Default values for hostname is "127.0.0.1", for schema is "tcp://".
// An example broker URI would look like: tcp://foobar.com:1204
func (this *tcpServerSpecial) AddRemoteServer(server string) error {
	if len(server) > 0 && server[0] == ':' {
		server = "127.0.0.1" + server
	}
	if !strings.Contains(server, "://") {
		server = "tcp://" + server
	}
	remoteURL, err := url.Parse(server)
	if err != nil {
		return err
	}
	this.server = remoteURL
	return nil
}

// Start start the server,and return quickly,if it nil,the server will connecting background,other failed
func (this *tcpServerSpecial) Start() error {
	if this.server == nil {
		return errors.New("empty remote server")
	}

	go this.run()
	return nil
}

// 增加间隔
func (this *tcpServerSpecial) run() {
	var ctx context.Context
	this.rwMux.Lock()
	if !atomic.CompareAndSwapUint32(&this.status, initial, disconnected) {
		this.rwMux.Unlock()
		return
	}
	ctx, this.cancel = context.WithCancel(context.Background())
	this.rwMux.Unlock()
	defer this.setConnectStatus(initial)

	for {
		select {
		case <-ctx.Done():
			return
		default:

		}

		this.Debug("connecting server %+v", this.server)
		conn, err := openConnection(this.server, this.TLSConfig, this.connectTimeout)
		if err != nil {
			this.Error("connect failed, %v", err)
			if !this.autoReconnect {
				return
			}
			time.Sleep(this.reconnectInterval)
			continue
		}
		this.Debug("connect success")
		this.conn = conn
		if err := this.onConnect(this); err != nil {
			time.Sleep(this.reconnectInterval)
			continue
		}
		this.setConnectStatus(connected)
		this.running(ctx)
		this.setConnectStatus(disconnected)
		this.onConnectionLost(this)
		select {
		case <-ctx.Done():
			return
		default:
			// 随机500ms-1s的重试，避免快速重试造成服务器许多无效连接
			time.Sleep(time.Millisecond * time.Duration(500+rand.Intn(5)))
		}
	}
}

// IsConnected check connect is online
func (this *tcpServerSpecial) IsConnected() bool {
	return this.connectStatus() == connected
}

func (this *tcpServerSpecial) IsClosed() bool {
	return this.connectStatus() == initial
}

// Close close the server
func (this *tcpServerSpecial) Close() error {
	this.rwMux.Lock()
	if this.cancel != nil {
		this.cancel()
	}
	this.rwMux.Unlock()
	return nil
}

func (this *tcpServerSpecial) setConnectStatus(status uint32) {
	this.rwMux.Lock()
	atomic.StoreUint32(&this.status, status)
	this.rwMux.Unlock()
}

func (this *tcpServerSpecial) connectStatus() uint32 {
	this.rwMux.RLock()
	status := atomic.LoadUint32(&this.status)
	this.rwMux.RUnlock()
	return status
}

func openConnection(uri *url.URL, tlsc *tls.Config, timeout time.Duration) (net.Conn, error) {
	switch uri.Scheme {
	case "tcp":
		return net.DialTimeout("tcp", uri.Host, timeout)
	case "ssl":
		fallthrough
	case "tls":
		fallthrough
	case "tcps":
		return tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", uri.Host, tlsc)
	}
	return nil, errors.New("Unknown protocol")
}
