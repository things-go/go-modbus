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
	DefaultKeepAliveInterval = 30 * time.Second
)
const (
	initial uint32 = iota
	disconnected
	connected
)

// OnConnectHandler when connected it will be call
type OnConnectHandler func(c TCPServerSpecial) error

// OnConnectionLostHandler when Connection lost it will be call
type OnConnectionLostHandler func(c TCPServerSpecial)

// KeepAlive keep alive function
type OnKeepAliveHandler func(c TCPServerSpecial)

// TCPServerSpecial modbus tcp server special
type TCPServerSpecial struct {
	ServerSession
	server    *url.URL // 连接的服务器端
	TLSConfig *tls.Config
	rwMux     sync.RWMutex
	status    uint32 // 状态

	connectTimeout    time.Duration           // 连接超时时间
	autoReconnect     bool                    // 是否启动重连
	reconnectInterval time.Duration           // 重连间隔时间
	enableKeepAlive   bool                    // 是否使能心跳包
	keepAliveInterval time.Duration           // 心跳包间隔
	onConnect         OnConnectHandler        // 连接回调
	onConnectionLost  OnConnectionLostHandler // 失连回调
	onKeepAlive       OnKeepAliveHandler      // 保活函数
	cancel            context.CancelFunc      // cancel
}

// NewTCPServerSpecial new tcp server special
func NewTCPServerSpecial() *TCPServerSpecial {
	return &TCPServerSpecial{
		ServerSession: ServerSession{
			readTimeout:  TCPDefaultReadTimeout,
			writeTimeout: TCPDefaultWriteTimeout,
			serverCommon: newServerCommon(),
			clogs:        NewClogWithPrefix("modbusTCPServerSpec =>"),
		},
		connectTimeout:    DefaultConnectTimeout,
		autoReconnect:     true,
		reconnectInterval: DefaultReconnectInterval,
		enableKeepAlive:   false,
		keepAliveInterval: DefaultKeepAliveInterval,
		onKeepAlive:       func(TCPServerSpecial) {},
		onConnect:         func(TCPServerSpecial) error { return nil },
		onConnectionLost:  func(TCPServerSpecial) {},
	}
}

// UnderlyingConn got underlying tcp conn
func (this *TCPServerSpecial) UnderlyingConn() net.Conn {
	return this.conn
}

// SetConnectTimeout set tcp connect the host timeout
func (this *TCPServerSpecial) SetConnectTimeout(t time.Duration) {
	this.connectTimeout = t
}

// SetReconnectInterval set tcp  reconnect the host interval when connect failed after try
func (this *TCPServerSpecial) SetReconnectInterval(t time.Duration) {
	this.reconnectInterval = t
}

func (this *TCPServerSpecial) EnableAutoReconnect(b bool) {
	this.autoReconnect = b
}

// SetTLSConfig set tls config
func (this *TCPServerSpecial) SetTLSConfig(t *tls.Config) {
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
func (this *TCPServerSpecial) SetOnConnectHandler(f OnConnectHandler) {
	if f != nil {
		this.onConnect = f
	}
}

// SetConnectionLostHandler set connection lost handler
func (this *TCPServerSpecial) SetConnectionLostHandler(f OnConnectionLostHandler) {
	if f != nil {
		this.onConnectionLost = f
	}
}
func (this *TCPServerSpecial) SetKeepAlive(b bool, t time.Duration, f OnKeepAliveHandler) {
	this.enableKeepAlive = b
	if t > 0 {
		this.keepAliveInterval = t
	}
	if f != nil {
		this.onKeepAlive = f
	}
}

// AddRemoteServer adds a broker URI to the list of brokers to be used.
// The format should be scheme://host:port
// Default values for hostname is "127.0.0.1", for schema is "tcp://".
// An example broker URI would look like: tcp://foobar.com:1204
func (this *TCPServerSpecial) AddRemoteServer(server string) error {
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
func (this *TCPServerSpecial) Start() error {
	if this.server == nil {
		return errors.New("empty remote server")
	}

	go this.run()
	return nil
}

// 增加间隔
func (this *TCPServerSpecial) run() {
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
		if this.enableKeepAlive {
			go func() {
				tick := time.NewTicker(this.keepAliveInterval)
				defer tick.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-tick.C:
						this.onKeepAlive(this)
					}
				}
			}()
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
func (this *TCPServerSpecial) IsConnected() bool {
	return this.connectStatus() == connected
}

// IsConnected check server is closed
func (this *TCPServerSpecial) IsClosed() bool {
	return this.connectStatus() == initial
}

// Close close the server
func (this *TCPServerSpecial) Close() error {
	this.rwMux.Lock()
	if this.cancel != nil {
		this.cancel()
	}
	this.rwMux.Unlock()
	return nil
}

func (this *TCPServerSpecial) setConnectStatus(status uint32) {
	this.rwMux.Lock()
	atomic.StoreUint32(&this.status, status)
	this.rwMux.Unlock()
}

func (this *TCPServerSpecial) connectStatus() uint32 {
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
