package modbus

import (
	"context"
	"crypto/tls"
	"errors"
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
type OnConnectHandler func(c *TCPServerSpecial) error

// OnConnectionLostHandler when Connection lost it will be call
type OnConnectionLostHandler func(c *TCPServerSpecial)

// OnKeepAliveHandler keep alive function
type OnKeepAliveHandler func(c *TCPServerSpecial)

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
			logger:       newLogger("modbusTCPServerSpec =>"),
		},
		connectTimeout:    DefaultConnectTimeout,
		autoReconnect:     true,
		reconnectInterval: DefaultReconnectInterval,
		enableKeepAlive:   false,
		keepAliveInterval: DefaultKeepAliveInterval,
		onKeepAlive:       func(*TCPServerSpecial) {},
		onConnect:         func(*TCPServerSpecial) error { return nil },
		onConnectionLost:  func(*TCPServerSpecial) {},
	}
}

// UnderlyingConn got underlying tcp conn
func (sf *TCPServerSpecial) UnderlyingConn() net.Conn {
	return sf.conn
}

// SetConnectTimeout set tcp connect the host timeout
func (sf *TCPServerSpecial) SetConnectTimeout(t time.Duration) {
	sf.connectTimeout = t
}

// SetReconnectInterval set tcp  reconnect the host interval when connect failed after try
func (sf *TCPServerSpecial) SetReconnectInterval(t time.Duration) {
	sf.reconnectInterval = t
}

// EnableAutoReconnect enable auto reconnect
func (sf *TCPServerSpecial) EnableAutoReconnect(b bool) {
	sf.autoReconnect = b
}

// SetTLSConfig set tls config
func (sf *TCPServerSpecial) SetTLSConfig(t *tls.Config) {
	sf.TLSConfig = t
}

// SetReadTimeout set read timeout
func (sf *ServerSession) SetReadTimeout(t time.Duration) {
	sf.readTimeout = t
}

// SetWriteTimeout set write timeout
func (sf *ServerSession) SetWriteTimeout(t time.Duration) {
	sf.writeTimeout = t
}

// SetOnConnectHandler set on connect handler
func (sf *TCPServerSpecial) SetOnConnectHandler(f OnConnectHandler) {
	if f != nil {
		sf.onConnect = f
	}
}

// SetConnectionLostHandler set connection lost handler
func (sf *TCPServerSpecial) SetConnectionLostHandler(f OnConnectionLostHandler) {
	if f != nil {
		sf.onConnectionLost = f
	}
}

// SetKeepAlive set keep alive enable, alive time and handler
func (sf *TCPServerSpecial) SetKeepAlive(b bool, t time.Duration, f OnKeepAliveHandler) {
	sf.enableKeepAlive = b
	if t > 0 {
		sf.keepAliveInterval = t
	}
	if f != nil {
		sf.onKeepAlive = f
	}
}

// AddRemoteServer adds a broker URI to the list of brokers to be used.
// The format should be scheme://host:port
// Default values for hostname is "127.0.0.1", for schema is "tcp://".
// An example broker URI would look like: tcp://foobar.com:1204
func (sf *TCPServerSpecial) AddRemoteServer(server string) error {
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
	sf.server = remoteURL
	return nil
}

// Start start the server,and return quickly,if it nil,the server will connecting background,other failed
func (sf *TCPServerSpecial) Start() error {
	if sf.server == nil {
		return errors.New("empty remote server")
	}

	go sf.run()
	return nil
}

// 增加间隔
func (sf *TCPServerSpecial) run() {
	var ctx context.Context

	sf.rwMux.Lock()
	if !atomic.CompareAndSwapUint32(&sf.status, initial, disconnected) {
		sf.rwMux.Unlock()
		return
	}
	ctx, sf.cancel = context.WithCancel(context.Background())
	sf.rwMux.Unlock()
	defer func() {
		sf.setConnectStatus(initial)
		sf.Debug("tcp server special stop!")
	}()
	sf.Debug("tcp server special start!")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sf.Debug("connecting server %+v", sf.server)
		conn, err := openConnection(sf.server, sf.TLSConfig, sf.connectTimeout)
		if err != nil {
			sf.Error("connect failed, %v", err)
			if !sf.autoReconnect {
				return
			}
			time.Sleep(sf.reconnectInterval)
			continue
		}
		sf.Debug("connect success")
		sf.conn = conn
		if err := sf.onConnect(sf); err != nil {
			time.Sleep(sf.reconnectInterval)
			continue
		}

		stopKeepAlive := make(chan struct{})
		if sf.enableKeepAlive {
			go func() {
				tick := time.NewTicker(sf.keepAliveInterval)
				defer tick.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-stopKeepAlive:
						return
					case <-tick.C:
						sf.onKeepAlive(sf)
					}
				}
			}()
		}
		sf.setConnectStatus(connected)
		sf.running(ctx)
		sf.setConnectStatus(disconnected)
		sf.onConnectionLost(sf)
		close(stopKeepAlive)
		select {
		case <-ctx.Done():
			return
		default:
			// 随机500ms-1s的重试，避免快速重试造成服务器许多无效连接
			time.Sleep(time.Millisecond * time.Duration(500+rand.Intn(500)))
		}
	}
}

// IsConnected check connect is online
func (sf *TCPServerSpecial) IsConnected() bool {
	return sf.connectStatus() == connected
}

// IsClosed check server is closed
func (sf *TCPServerSpecial) IsClosed() bool {
	return sf.connectStatus() == initial
}

// Close close the server
func (sf *TCPServerSpecial) Close() error {
	sf.rwMux.Lock()
	if sf.cancel != nil {
		sf.cancel()
	}
	sf.rwMux.Unlock()
	return nil
}

func (sf *TCPServerSpecial) setConnectStatus(status uint32) {
	sf.rwMux.Lock()
	atomic.StoreUint32(&sf.status, status)
	sf.rwMux.Unlock()
}

func (sf *TCPServerSpecial) connectStatus() uint32 {
	sf.rwMux.RLock()
	status := atomic.LoadUint32(&sf.status)
	sf.rwMux.RUnlock()
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
