package modbus

import (
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
	addr            string
	tcpReadTimeout  time.Duration
	tcpWriteTimeout time.Duration
	node            sync.Map
	*pool
	mu     sync.Mutex
	listen net.Listener
	client map[net.Conn]struct{}
	wg     sync.WaitGroup
	*serverHandler
	clogs
}

// NewTCPServer the modbus server listening on "address:port".
func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{
		addr:            addr,
		tcpReadTimeout:  TCPDefaultReadTimeout,
		tcpWriteTimeout: TCPDefaultWriteTimeout,
		pool:            newPool(tcpAduMaxSize),
		serverHandler:   newServerHandler(),
		client:          make(map[net.Conn]struct{}),
		clogs:           clogs{newDefaultLogger("modbusTCPSlave =>"), 0},
	}
}

// AddNodes 增加节点
func (this *TCPServer) AddNodes(nodes ...*NodeRegister) {
	for _, v := range nodes {
		this.node.Store(v.slaveID, v)
	}
}

// DeleteNode 删除一个节点
func (this *TCPServer) DeleteNode(slaveID byte) {
	this.node.Delete(slaveID)
}

// DeleteAllNode 删除所有节点
func (this *TCPServer) DeleteAllNode() {
	this.node.Range(func(k, v interface{}) bool {
		this.node.Delete(k)
		return true
	})
}

// GetNode 获取一个节点
func (this *TCPServer) GetNode(slaveID byte) (*NodeRegister, error) {
	v, ok := this.node.Load(slaveID)
	if !ok {
		return nil, errors.New("slaveID not exist")
	}
	return v.(*NodeRegister), nil
}

// GetNodeList 获取节点列表
func (this *TCPServer) GetNodeList() []*NodeRegister {
	list := make([]*NodeRegister, 0)
	this.node.Range(func(k, v interface{}) bool {
		list = append(list, v.(*NodeRegister))
		return true
	})
	return list
}

// Range 扫描节点 same as sync map range
func (this *TCPServer) Range(f func(slaveID byte, node *NodeRegister) bool) {
	this.node.Range(func(k, v interface{}) bool {
		return f(k.(byte), v.(*NodeRegister))
	})
}

// ServerModbus 服务
func (this *TCPServer) ServerModbus() {
	listen, err := net.Listen("tcp", this.addr)
	if err != nil {
		this.Error("modbus server: %v\n", err)
		return
	}
	this.mu.Lock()
	this.listen = listen
	this.mu.Unlock()
	defer this.Close()
	this.Debug("server running")
	for {
		conn, err := listen.Accept()
		if err != nil {
			this.Error("accept failed, %v\n", err)
			return
		}
		this.wg.Add(1)
		go func() {
			this.Debug("client(%v) -> server(%v) connected", conn.RemoteAddr(), conn.LocalAddr())
			this.mu.Lock()
			this.client[conn] = struct{}{}
			this.mu.Unlock()

			if err := this.handlerModbus(conn); err != nil {
				this.Error("handler modbus failed %v", err)
			}

			this.Debug("client(%v) -> server(%v) disconnected", conn.RemoteAddr(), conn.LocalAddr())
			this.mu.Lock()
			delete(this.client, conn)
			this.mu.Unlock()
			conn.Close()
			this.wg.Done()
		}()
	}
}

// handler net conn
func (this *TCPServer) handlerModbus(conn net.Conn) error {
	// get pool frame
	frame := this.pool.get()
	defer func() {
		this.pool.put(frame)
	}()

	for {
		adu := frame.adu[:tcpAduMaxSize]
		for length, rdCnt := tcpHeaderMbapSize, 0; rdCnt < length; {
			err := conn.SetReadDeadline(time.Now().Add(this.tcpReadTimeout))
			if err != nil {
				return fmt.Errorf("set read deadline %v", err)
			}
			bytesRead, err := io.ReadFull(conn, adu[rdCnt:length])
			if err != nil {
				if err != io.EOF && err != io.ErrClosedPipe || strings.Contains(err.Error(), "use of closed network connection") {
					return err
				}

				if e, ok := err.(net.Error); ok && !e.Temporary() {
					return err
				}

				if bytesRead == 0 && err == io.EOF {
					return fmt.Errorf("remote client closed %v", err)
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
					this.Debug("RX Raw[% x]", adu[:length])
					response, err := this.frameHandler(adu[:length])
					if err != nil {
						return fmt.Errorf("frameHandler %v", err)
					}
					this.Debug("TX Raw[% x]", response)

					err = func(b []byte) error {
						for wrCnt := 0; len(b) > wrCnt; {
							err = conn.SetWriteDeadline(time.Now().Add(this.tcpWriteTimeout))
							if err != nil {
								return fmt.Errorf("set read deadline %v", err)
							}
							byteCount, err := conn.Write(b[wrCnt:])
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
					}(response)
					if err != nil {
						return err
					}
				}
			}
		}
	}
}

// Close close the server
func (this *TCPServer) Close() error {
	this.mu.Lock()
	if this.listen != nil {
		this.listen.Close()
		this.listen = nil
	}
	for k := range this.client {
		k.Close()
	}
	this.mu.Unlock()
	this.wg.Wait()
	return nil
}

// modbus 包处理
func (this *TCPServer) frameHandler(requestAdu []byte) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			this.Error("painc happen,%v", err)
		}
	}()

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
	if err != nil {
		return nil, err
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

	return responseAdu, nil
}
