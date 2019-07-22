package modbus

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

// defined default value	io.Closer
const (
	DefaultConnectTimeout    = 15 * time.Second
	DefaultReconnectInterval = 1 * time.Minute
)

// TCPServerSpecial server special interface
type TCPServerSpecial interface {
	UnderlyingConn() net.Conn
	IsConnected() bool
	Start() error
	io.Closer

	SetTLSConfig(t *tls.Config)
	AddRemoteServer(server string) error
	SetOnConnectHandler(f OnConnectHandler)
	SetConnectionLostHandler(f ConnectionLostHandler)

	LogMode(enable bool)
	SetLogProvider(p LogProvider)
}

// OnConnectHandler when connected it will be call
type OnConnectHandler func(c TCPServerSpecial)

// ConnectionLostHandler when Connection lost it will be call
type ConnectionLostHandler func(c TCPServerSpecial)

// tcpServerSpecial modbus tcp server special
type tcpServerSpecial struct {
	ServerSession
	Server            *url.URL      // 连接的服务器端
	connectTimeout    time.Duration // 连接超时时间
	autoReconnect     bool          // 是否启动重连
	ReconnectInterval time.Duration // 重连间隔时间
	TLSConfig         *tls.Config
	onConnect         OnConnectHandler
	onConnectionLost  ConnectionLostHandler
}

func NewTCPServerSpecial() *tcpServerSpecial {
	return &tcpServerSpecial{
		ServerSession: ServerSession{
			readTimeout:  TCPDefaultReadTimeout,
			writeTimeout: TCPDefaultWriteTimeout,
			serverCommon: newServerCommon(),
			clogs:        NewClogWithPrefix("modbus serverSpec =>"),
		},
		connectTimeout:    DefaultConnectTimeout,
		autoReconnect:     true,
		ReconnectInterval: DefaultReconnectInterval,
		onConnect:         func(TCPServerSpecial) {},
		onConnectionLost:  func(TCPServerSpecial) {},
	}
}
func (this *tcpServerSpecial) UnderlyingConn() net.Conn {
	return this.conn
}

// SetConnectTimeout set tcp connect the host timeout
func (this *tcpServerSpecial) SetConnectTimeout(t time.Duration) {
	this.connectTimeout = t
}

// SetReconnectInterval set tcp  reconnect the host interval when connect failed after try
func (this *tcpServerSpecial) SetReconnectInterval(t time.Duration) {
	this.ReconnectInterval = t
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
	this.Server = remoteURL
	return nil
}

func (this *tcpServerSpecial) Start() error {
	if this.Server == nil {
		return errors.New("empty remote server")
	}

	go this.running()
	return nil
}

// 增加30秒 重连间隔
func (this *tcpServerSpecial) running() {
	//this.rwMux.Lock()
	//if !atomic.CompareAndSwapUint32(&this.status, disconnected, connecting) {
	//	this.rwMux.Unlock()
	//	return
	//}
	//this.rwMux.Unlock()

	for {
		this.Debug("connecting server %+v", this.Server)
		conn, err := openConnection(this.Server, this.TLSConfig, this.connectTimeout)
		if err != nil {
			this.Error("connect failed, %v", err)
			if !this.autoReconnect {
				//			this.setConnectStatus(disconnected)
				return
			}
			time.Sleep(this.ReconnectInterval)
			continue
		}
		this.Debug("connect success")
		this.conn = conn
		this.onConnect(this)
		//		this.run(context.Background())
		this.onConnectionLost(this)

	}
}

func (this *tcpServerSpecial) IsConnected() bool {
	return false
}

func (this *tcpServerSpecial) Close() error {
	return nil
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
