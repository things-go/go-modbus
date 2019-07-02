package modbus

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/thinkgos/library/elog"
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
	pool            *sync.Pool
	mu              sync.Mutex
	listen          net.Listener
	client          map[net.Conn]struct{}
	wg              sync.WaitGroup
	*serverHandler
	logs
}

// NewTCPServer the modbus server listening on "address:port".
func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{
		addr:            addr,
		tcpReadTimeout:  TCPDefaultReadTimeout,
		tcpWriteTimeout: TCPDefaultWriteTimeout,
		pool:            &sync.Pool{New: func() interface{} { return &protocolTCPFrame{} }},
		serverHandler:   newServerHandler(),
		client:          make(map[net.Conn]struct{}),
		logs: logs{
			Elog: elog.NewElog(nil),
		},
	}
}

// AddNode 增加节点
func (this *TCPServer) AddNode(node *NodeRegister) {
	if node != nil {
		this.node.Store(node.slaveID, node)
	}

}

// AddNodes 批量增加节点
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
		this.Error("mobus listen: %v\n", err)
		return
	}
	this.mu.Lock()
	this.listen = listen
	this.mu.Unlock()
	defer this.Close()
	this.Debug("mobus TCP server running")
	for {
		conn, err := listen.Accept()
		if err != nil {
			this.Error("modbus accept: %#v\n", err)
			return
		}
		this.wg.Add(1)
		go func() {
			this.Debug("client(%v) -> server(%v) connected", conn.RemoteAddr(), conn.LocalAddr())
			// get pool frame
			frame := this.pool.Get().(*protocolTCPFrame)
			this.mu.Lock()
			this.client[conn] = struct{}{}
			this.mu.Unlock()
			defer func() {
				this.Debug("client(%v) -> server(%v) disconnected", conn.RemoteAddr(), conn.LocalAddr())
				// rest pool frame and put it
				frame.pdu.Data = nil
				this.pool.Put(frame)

				this.mu.Lock()
				delete(this.client, conn)
				this.mu.Unlock()
				conn.Close()
				this.wg.Done()
			}()

			for {
				adu := frame.adu[:]
				length := tcpHeaderMbapSize
				for rdCnt := 0; rdCnt < length; {
					err := conn.SetReadDeadline(time.Now().Add(this.tcpReadTimeout))
					if err != nil {
						this.Error("set read deadline %v\n", err)
						return
					}
					bytesRead, err := io.ReadFull(conn, frame.adu[rdCnt:length])
					if err != nil {
						if err != io.EOF && err != io.ErrClosedPipe || strings.Contains(err.Error(), "use of closed network connection") {
							this.Error("modbus server: %v", err)
							return
						}

						if e, ok := err.(net.Error); ok && !e.Temporary() {
							this.Error("modbus server: %v", err)
							return
						}

						if bytesRead == 0 && err == io.EOF {
							this.Error("remote client closed %v\n", err)
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
							this.Debug("modbus request: % x", adu[:length])
							response, err := this.frameHandler(frame, adu[:length])
							if err != nil {
								this.Error("modbus handler: %v", err)
								break
							}
							this.Debug("modbus response: % x", response)

							err = func(b []byte) error {
								for wrCnt := 0; len(b) > wrCnt; {
									err = conn.SetWriteDeadline(time.Now().Add(this.tcpWriteTimeout))
									if err != nil {
										this.Error("set read deadline %v\n", err)
										return err
									}
									byteCount, err := conn.Write(b[wrCnt:])
									if err != nil {
										// See: https://github.com/golang/go/issues/4373
										if err != io.EOF && err != io.ErrClosedPipe || strings.Contains(err.Error(), "use of closed network connection") {
											this.Error("cs104 server: %v", err)
											return err
										}
										if e, ok := err.(net.Error); !ok || !e.Temporary() {
											this.Error("cs104 server: %v", err)
											return err
										}
										// temporary error may be recoverable
									}
									wrCnt += byteCount
								}
								return nil
							}(response)
							if err != nil {
								this.Error("modbus write: %v", err)
								return
							}
						}
					}
				}
			}
		}()
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
func (this *TCPServer) frameHandler(frame *protocolTCPFrame, requestAdu []byte) ([]byte, error) {
	// copy head from request adu
	frame.head.transactionID = binary.BigEndian.Uint16(requestAdu[0:])
	frame.head.protocolID = binary.BigEndian.Uint16(requestAdu[2:])
	frame.head.length = binary.BigEndian.Uint16(requestAdu[4:])
	frame.head.slaveID = uint8(requestAdu[6])
	frame.pdu.FuncCode = uint8(requestAdu[7])
	pduData := requestAdu[8:]

	node, err := this.GetNode(frame.head.slaveID)
	if err != nil {
		return nil, err
	}
	if handle, ok := this.function[frame.pdu.FuncCode]; ok {
		frame.pdu.Data, err = handle(node, pduData)
	} else {
		err = &ExceptionError{ExceptionCodeIllegalFunction}
	}
	if err != nil {
		frame.pdu.FuncCode |= 0x80
		frame.pdu.Data = []byte{err.(*ExceptionError).ExceptionCode}
	}

	// prepare response data,fill it
	frame.head.length = uint16(2 + len(frame.pdu.Data))
	response := frame.adu[:tcpHeaderMbapSize+1+len(frame.pdu.Data)]
	binary.BigEndian.PutUint16(response[0:], frame.head.transactionID)
	binary.BigEndian.PutUint16(response[2:], frame.head.protocolID)
	binary.BigEndian.PutUint16(response[4:], frame.head.length)
	response[6] = frame.head.slaveID
	response[7] = frame.pdu.FuncCode
	copy(response[8:], frame.pdu.Data)

	return response, nil
}
