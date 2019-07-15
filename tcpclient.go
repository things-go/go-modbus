package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// TCPDefaultTimeout TCP Default timeout
	TCPDefaultTimeout = 1 * time.Second
	// TCPDefaultAutoReconnect TCP Default auto reconnect count
	TCPDefaultAutoReconnect = 1
)

// TCPClientProvider implements ClientProvider interface.
type TCPClientProvider struct {
	logs
	Address string
	mu      sync.Mutex
	// TCP connection
	conn net.Conn
	// Connect & Read timeout
	Timeout time.Duration
	// if > 0, when disconnect,it will try to reconnect the remote
	// but if we active close self,it will not to reconncet
	// if == 0 auto reconnect not active
	autoReconnect byte
	// For synchronization between messages of server & client
	transactionID uint32
	// 请求池,所有tcp客户端共用一个请求池
	pool *sync.Pool
}

// check TCPClientProvider implements underlying method
var _ ClientProvider = (*TCPClientProvider)(nil)

// 请求池,所有TCP客户端共用一个请求池
var tcpPool = &sync.Pool{New: func() interface{} { return &protocolTCPFrame{} }}

// NewTCPClientProvider allocates a new TCPClientProvider.
func NewTCPClientProvider(address string) *TCPClientProvider {
	return &TCPClientProvider{
		Address:       address,
		Timeout:       TCPDefaultTimeout,
		autoReconnect: TCPDefaultAutoReconnect,
		pool:          tcpPool,
		logs:          logs{newLogger(), 0},
	}
}

// encode modbus application protocol header & pdu to TCP frame,return adu
//  ---- MBAP header ----
//  Transaction identifier: 2 bytes
//  Protocol identifier: 2 bytes
//  Length: 2 bytes
//  Unit identifier: 1 byte
//  ---- data Unit ----
//  Function code: 1 byte
//  Data: n bytes
func (this *protocolTCPFrame) encode(slaveID byte, pdu *ProtocolDataUnit) ([]byte, error) {
	length := tcpHeaderMbapSize + 1 + len(pdu.Data)
	if length > tcpAduMaxSize {
		return nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'", length, tcpAduMaxSize)
	}
	this.pdu.FuncCode = pdu.FuncCode
	this.head.length = uint16(2 + len(pdu.Data)) // Length = sizeof(SlaveId) + sizeof(FuncCode) + Data
	this.head.slaveID = slaveID
	this.head.protocolID = tcpProtocolIdentifier

	// fill adu buffer
	adu := this.adu[0:length]
	binary.BigEndian.PutUint16(adu, this.head.transactionID)  // MBAP Transaction identifier
	binary.BigEndian.PutUint16(adu[2:], this.head.protocolID) // MBAP Protocol identifier
	binary.BigEndian.PutUint16(adu[4:], this.head.length)     // MBAP Length
	adu[6] = this.head.slaveID                                // MBAP Unit identifier
	adu[tcpHeaderMbapSize] = this.pdu.FuncCode                // PDU funcCode
	copy(adu[tcpHeaderMbapSize+1:], pdu.Data)                 // PDU data
	return adu, nil
}

// decode extracts tcpHeader & PDU from TCP frame:
//  ---- MBAP header ----
//  Transaction identifier: 2 bytes
//  Protocol identifier: 2 bytes
//  Length: 2 bytes
//  Unit identifier: 1 byte
//  ---- data Unit ----
//  Function        : 1 byte
//  Data            : 0 up to 252 bytes
func decodeTCPFrame(adu []byte) (*protocolTCPHeader, []byte, error) {
	if len(adu) < tcpAduMinSize { // Minimum size (including MBAP, funcCode)
		return nil, nil, fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'", len(adu), tcpAduMinSize)
	}
	// Read length value in the header
	head := &protocolTCPHeader{
		transactionID: binary.BigEndian.Uint16(adu),
		protocolID:    binary.BigEndian.Uint16(adu[2:]),
		length:        binary.BigEndian.Uint16(adu[4:]),
		slaveID:       adu[6],
	}

	pduLength := len(adu) - tcpHeaderMbapSize
	if pduLength != int(head.length-1) {
		return nil, nil, fmt.Errorf("modbus: length in response '%v' does not match pdu data length '%v'",
			head.length-1, pduLength)

	}
	// The first byte after header is function code
	return head, adu[tcpHeaderMbapSize:], nil
}

// verify confirms valid data
func (this *protocolTCPFrame) verify(rspHead *protocolTCPHeader, rspPDU *ProtocolDataUnit) error {
	switch {
	case rspHead.transactionID != this.head.transactionID:
		// Check transaction ID
		return fmt.Errorf("modbus: response transaction id '%v' does not match request '%v'",
			rspHead.transactionID, this.head.transactionID)
	case rspHead.protocolID != this.head.protocolID:
		// Check protocol ID
		return fmt.Errorf("modbus: response protocol id '%v' does not match request '%v'",
			rspHead.protocolID, this.head.protocolID)
	case rspHead.slaveID != this.head.slaveID:
		// Check slaveID same
		return fmt.Errorf("modbus: response unit id '%v' does not match request '%v'",
			rspHead.slaveID, this.head.slaveID)
	case rspPDU.FuncCode != this.pdu.FuncCode:
		// Check correct function code returned (exception)
		return responseError(rspPDU)
	case rspPDU.Data == nil || len(rspPDU.Data) == 0:
		// check Empty response
		return fmt.Errorf("modbus: response data is empty")
	}
	return nil
}

// Send the request to tcp and get the response
func (this *TCPClientProvider) Send(slaveID byte, request *ProtocolDataUnit) (*ProtocolDataUnit, error) {
	frame := this.pool.Get().(*protocolTCPFrame)
	defer this.pool.Put(frame)
	// add transaction id
	frame.head.transactionID = uint16(atomic.AddUint32(&this.transactionID, 1))

	aduRequest, err := frame.encode(slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := this.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspHead, pdu, err := decodeTCPFrame(aduResponse)
	if err != nil {
		return nil, err
	}
	response := &ProtocolDataUnit{pdu[0], pdu[1:]}
	if err = frame.verify(rspHead, response); err != nil {
		return nil, err
	}
	return response, nil
}

// SendPdu send pdu request to the remote server
func (this *TCPClientProvider) SendPdu(slaveID byte, pduRequest []byte) (pduResponse []byte, err error) {
	if len(pduRequest) < pduMinSize || len(pduRequest) > pduMaxSize {
		return nil, fmt.Errorf("modbus: rspPdu size '%v' must not be between '%v' and '%v'",
			len(pduRequest), pduMinSize, pduMaxSize)
	}

	frame := this.pool.Get().(*protocolTCPFrame)
	defer this.pool.Put(frame)
	// add transaction id
	frame.head.transactionID = uint16(atomic.AddUint32(&this.transactionID, 1))

	request := &ProtocolDataUnit{pduRequest[0], pduRequest[1:]}
	aduRequest, err := frame.encode(slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := this.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspHead, rspPdu, err := decodeTCPFrame(aduResponse)
	if err != nil {
		return nil, err
	}
	response := &ProtocolDataUnit{rspPdu[0], rspPdu[1:]}
	if err = frame.verify(rspHead, response); err != nil {
		return nil, err
	}
	// rspPdu pass tcpMBAP head
	return rspPdu, nil
}

// SendRawFrame send raw adu request frame
func (this *TCPClientProvider) SendRawFrame(aduRequest []byte) (aduResponse []byte, err error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if !this.isConnected() {
		return nil, ErrClosedConnection
	}
	// Send data
	this.Debug("modbus: sending % x", aduRequest)
	// Set write and read timeout
	var timeout time.Time
	var tryCnt byte
	for {
		if this.Timeout > 0 {
			timeout = time.Now().Add(this.Timeout)
		}
		if err = this.conn.SetDeadline(timeout); err != nil {
			return nil, err
		}

		_, err = this.conn.Write(aduRequest)
		if err == nil { // success
			break
		}
		if this.autoReconnect == 0 {
			return
		}
		for {
			err = this.connect()
			if err == nil {
				break
			}
			if tryCnt++; tryCnt >= this.autoReconnect {
				return
			}
		}
	}

	// Read header first
	var data [tcpAduMaxSize]byte
	if _, err = io.ReadFull(this.conn, data[:tcpHeaderMbapSize]); err != nil {
		return
	}
	// Read length, ignore transaction & protocol id (4 bytes)
	length := int(binary.BigEndian.Uint16(data[4:]))
	switch {
	case length <= 0:
		_ = this.flush(data[:])
		err = fmt.Errorf("modbus: length in response header '%v' must not be zero", length)
		return
	case length > (tcpAduMaxSize - (tcpHeaderMbapSize - 1)):
		_ = this.flush(data[:])
		err = fmt.Errorf("modbus: length in response header '%v' must not greater than '%v'", length, tcpAduMaxSize-tcpHeaderMbapSize+1)
		return
	}
	// Skip unit id
	length += tcpHeaderMbapSize - 1
	if _, err = io.ReadFull(this.conn, data[tcpHeaderMbapSize:length]); err != nil {
		return
	}
	aduResponse = data[:length]
	this.Debug("modbus: received % x\n", aduResponse)
	return
}

// Connect establishes a new connection to the address in Address.
// Connect and Close are exported so that multiple requests can be done with one session
func (this *TCPClientProvider) Connect() error {
	this.mu.Lock()
	err := this.connect()
	this.mu.Unlock()
	return err
}

// Caller must hold the mutex before calling this method.
func (this *TCPClientProvider) connect() error {
	dialer := &net.Dialer{Timeout: this.Timeout}
	conn, err := dialer.Dial("tcp", this.Address)
	if err != nil {
		return err
	}
	this.conn = conn
	return nil
}

// IsConnected returns a bool signifying whether
// the client is connected or not.
func (this *TCPClientProvider) IsConnected() bool {
	this.mu.Lock()
	b := this.isConnected()
	this.mu.Unlock()
	return b
}

// Caller must hold the mutex before calling this method.
func (this *TCPClientProvider) isConnected() bool {
	return this.conn != nil
}

// SetAutoReconnect set auto reconnect  retry count
func (this *TCPClientProvider) SetAutoReconnect(cnt byte) {
	this.mu.Lock()
	this.autoReconnect = cnt
	if this.autoReconnect > 6 {
		this.autoReconnect = 6
	}
	this.mu.Unlock()
}

// Close closes current connection.
func (this *TCPClientProvider) Close() error {
	var err error
	this.mu.Lock()
	if this.conn != nil {
		err = this.conn.Close()
		this.conn = nil
	}
	this.mu.Unlock()
	return err
}

// flush flushes pending data in the connection,
// returns io.EOF if connection is closed.
func (this *TCPClientProvider) flush(b []byte) (err error) {
	if err = this.conn.SetReadDeadline(time.Now()); err != nil {
		return
	}
	// Timeout setting will be reset when reading
	if _, err = this.conn.Read(b); err != nil {
		// Ignore timeout error
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			err = nil
		}
	}
	return
}
