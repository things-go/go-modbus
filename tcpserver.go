package modbus

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
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
	lAddr           string
	tcpReadTimeout  time.Duration
	tcpWriteTimeout time.Duration
	node            sync.Map
	pool            *sync.Pool
	mu              sync.Mutex
	listen          net.Listener
	client          map[net.Conn]struct{}
	*serverHandler
	logs
}

// NewTCPServer the Modbus server listening on "address:port".
func NewTCPServer(laddr string) *TCPServer {
	return &TCPServer{
		lAddr:           laddr,
		tcpReadTimeout:  TCPDefaultReadTimeout,
		tcpWriteTimeout: TCPDefaultWriteTimeout,
		pool:            &sync.Pool{New: func() interface{} { return &protocolTCPFrame{} }},
		serverHandler:   newServerHandler(),
		client:          make(map[net.Conn]struct{}),
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

// GetNode 获取一个节点
func (this *TCPServer) GetNode(slaveID byte) (*NodeRegister, error) {
	v, ok := this.node.Load(slaveID)
	if !ok {
		return nil, errors.New("slaveID not exist")
	}
	return v.(*NodeRegister), nil
}

// ServerModbus 服务
func (this *TCPServer) ServerModbus() {
	listen, err := net.Listen("tcp", this.lAddr)
	if err != nil {
		this.logf("mobus listen: %v\n", err)
		return
	}
	this.mu.Lock()
	this.listen = listen
	this.mu.Unlock()
	defer this.Close()
	this.logf("mobus TCP server running")
	for {
		conn, err := listen.Accept()
		if err != nil {
			this.logf("modbus accept: %#v\n", err)
			return
		}
		go func() {
			//this.logf("client(%v) disconnected", conn.RemoteAddr())
			log.Printf("client(%v) -> server(%v) connected", conn.RemoteAddr(), conn.LocalAddr())
			// get pool frame
			frame := this.pool.Get().(*protocolTCPFrame)
			this.mu.Lock()
			this.client[conn] = struct{}{}
			this.mu.Unlock()
			defer func() {
				//this.logf("client(%v) disconnected", conn.RemoteAddr())
				log.Printf("client(%v) -> server(%v) disconnected", conn.RemoteAddr(), conn.LocalAddr())
				// rest pool frame and put it
				frame.pdu.Data = nil
				this.pool.Put(frame)
				this.mu.Lock()
				delete(this.client, conn)
				this.mu.Unlock()
				conn.Close()
			}()
			readbuf := make([]byte, 1024)
			tmpbuf := make([]byte, 0, 512)
			for {
				err := conn.SetReadDeadline(time.Now().Add(this.tcpReadTimeout))
				if err != nil {
					this.logf("set read deadline %v\n", err)
					return
				}
				bytesRead, err := conn.Read(readbuf)
				if err != nil {
					if err != io.EOF {
						this.logf("read failed %v\n", err)
						return
					}
					// cnt >0 do nothing
					// cnt == continue next do it
				}
				if bytesRead == 0 {
					continue
				}

				tmpbuf = append(tmpbuf, readbuf[:bytesRead]...)

				for {
					if len(tmpbuf) < tcpHeaderMbapSize {
						break
					}
					// check head ProtocolIdentifier
					if binary.BigEndian.Uint16(tmpbuf[2:]) != tcpProtocolIdentifier {
						tmpbuf = tmpbuf[tcpHeaderMbapSize:]
						continue
					}
					// check buffer has enough bytes to read
					aduLenth := binary.BigEndian.Uint16(tmpbuf[4:]) + tcpHeaderMbapSize - 1
					if len(tmpbuf) < int(aduLenth) {
						break
					}
					request := tmpbuf[:aduLenth] // get request
					tmpbuf = tmpbuf[aduLenth:]   // past request
					// Set the length of the packet to the number of read bytes.
					this.logf("modbus request: % x", request)
					response, err := this.frameHandler(frame, request)
					if err != nil {
						this.logf("modbus handler: %v", err)
						continue
					}
					this.logf("modbus response: % x", response)
					err = conn.SetWriteDeadline(time.Now().Add(this.tcpWriteTimeout))
					if err != nil {
						this.logf("set read deadline %v\n", err)
						return
					}
					_, err = conn.Write(response)
					if err != nil {
						this.logf("modbus write: %v", err)
						return
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
	for k, _ := range this.client {
		k.Close()
		delete(this.client, k)
	}
	this.mu.Unlock()
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
