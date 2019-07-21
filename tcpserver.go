package modbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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
	readTimeout  time.Duration
	writeTimeout time.Duration
	*pool
	mu     sync.Mutex
	listen net.Listener
	wg     sync.WaitGroup
	cancel context.CancelFunc
	*serverCommon
	clogs
}

// NewTCPServer the modbus server listening on "address:port".
func NewTCPServer() *TCPServer {
	return &TCPServer{
		readTimeout:  TCPDefaultReadTimeout,
		writeTimeout: TCPDefaultWriteTimeout,
		pool:         newPool(tcpAduMaxSize),
		serverCommon: newServerHandler(),
		clogs:        clogs{newDefaultLogger("modbusTCPSlave =>"), 0},
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
			sess := &session{
				conn,
				this.readTimeout,
				this.writeTimeout,
				this.pool,
				this.serverCommon,
				&this.clogs,
			}
			sess.run(ctx)
			this.wg.Done()
		}()
	}
}

type session struct {
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	*pool
	*serverCommon
	*clogs
}

// handler net conn
func (this *session) run(ctx context.Context) {
	var err error
	var bytesRead int

	this.Debug("client(%v) -> server(%v) connected", this.conn.RemoteAddr(), this.conn.LocalAddr())
	// get pool frame
	frame := this.pool.get()
	defer func() {
		this.pool.put(frame)
		this.conn.Close()
		this.Debug("client(%v) -> server(%v) disconnected,cause by %v", this.conn.RemoteAddr(), this.conn.LocalAddr(), err)
	}()

	for {
		select {
		case <-ctx.Done():
			err = errors.New("server active close")
			return
		default:
		}

		adu := frame.adu[:tcpAduMaxSize]
		for length, rdCnt := tcpHeaderMbapSize, 0; rdCnt < length; {
			err = this.conn.SetReadDeadline(time.Now().Add(this.readTimeout))
			if err != nil {
				return
			}
			bytesRead, err = io.ReadFull(this.conn, adu[rdCnt:length])
			if err != nil {
				if err != io.EOF && err != io.ErrClosedPipe || strings.Contains(err.Error(), "use of closed network connection") {
					return
				}

				if e, ok := err.(net.Error); ok && !e.Temporary() {
					return
				}

				if bytesRead == 0 && err == io.EOF {
					err = fmt.Errorf("remote client closed, %v", err)
					return
				}
				// cnt >0 do nothing
				// cnt == 0 && err != io.EOFcontinue do it next
			}
			rdCnt += bytesRead
			if rdCnt >= length {
				// check hed ProtocolIdentifier
				if binary.BigEndian.Uint16(adu[2:]) != tcpProtocolIdentifier {
					break
				}
				length = int(binary.BigEndian.Uint16(adu[4:])) + tcpHeaderMbapSize - 1
				if rdCnt == length {
					err = this.frameHandler(adu[:length])
					if err != nil {
						return
					}
				}
			}
		}
	}
}

// modbus 包处理
func (this *session) frameHandler(requestAdu []byte) error {
	defer func() {
		if err := recover(); err != nil {
			this.Error("painc happen,%v", err)
		}
	}()
	this.Debug("RX Raw[% x]", requestAdu)
	// got head from request adu
	tcpHeader := protocolTCPHeader{
		binary.BigEndian.Uint16(requestAdu[0:]),
		binary.BigEndian.Uint16(requestAdu[2:]),
		binary.BigEndian.Uint16(requestAdu[4:]),
		uint8(requestAdu[6]),
	}
	funcCode := uint8(requestAdu[7])
	pduData := requestAdu[8:]

	node, err := this.GetNode(tcpHeader.slaveID)
	if err != nil { // slave id not exit, ignore it
		return nil
	}
	var rspPduData []byte
	if handle, ok := this.function[funcCode]; ok {
		rspPduData, err = handle(node, pduData)
	} else {
		err = &ExceptionError{ExceptionCodeIllegalFunction}
	}
	if err != nil {
		funcCode |= 0x80
		rspPduData = []byte{err.(*ExceptionError).ExceptionCode}
	}

	// prepare responseAdu data,fill it
	responseAdu := requestAdu[:tcpHeaderMbapSize]
	binary.BigEndian.PutUint16(responseAdu[0:], tcpHeader.transactionID)
	binary.BigEndian.PutUint16(responseAdu[2:], tcpHeader.protocolID)
	binary.BigEndian.PutUint16(responseAdu[4:], uint16(2+len(rspPduData)))
	responseAdu[6] = tcpHeader.slaveID
	responseAdu = append(responseAdu, funcCode)
	responseAdu = append(responseAdu, rspPduData...)

	this.Debug("TX Raw[% x]", responseAdu)

	return func(b []byte) error {
		for wrCnt := 0; len(b) > wrCnt; {
			err = this.conn.SetWriteDeadline(time.Now().Add(this.writeTimeout))
			if err != nil {
				return fmt.Errorf("set read deadline %v", err)
			}
			byteCount, err := this.conn.Write(b[wrCnt:])
			if err != nil {
				// See: https://github.com/golang/go/issues/4373
				if err != io.EOF && err != io.ErrClosedPipe ||
					strings.Contains(err.Error(), "use of closed network connection") {
					return err
				}
				if e, ok := err.(net.Error); !ok || !e.Temporary() {
					return err
				}
				// temporary error may be recoverable
			}
			wrCnt += byteCount
		}
		return nil
	}(responseAdu)
}
