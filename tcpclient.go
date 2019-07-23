package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
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
	clogs
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
	*pool
}

// check TCPClientProvider implements underlying method
var _ ClientProvider = (*TCPClientProvider)(nil)

// 请求池,所有TCP客户端共用一个请求池
var tcpPool = newPool(tcpAduMaxSize)

// NewTCPClientProvider allocates a new TCPClientProvider.
func NewTCPClientProvider(address string) *TCPClientProvider {
	return &TCPClientProvider{
		Address:       address,
		Timeout:       TCPDefaultTimeout,
		autoReconnect: TCPDefaultAutoReconnect,
		pool:          tcpPool,
		clogs:         clogs{newDefaultLogger("modbusTCPMaster =>"), 0},
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
func (this *protocolFrame) encodeTCPFrame(tid uint16, slaveID byte, pdu ProtocolDataUnit) (protocolTCPHeader, []byte, error) {
	length := tcpHeaderMbapSize + 1 + len(pdu.Data)
	if length > tcpAduMaxSize {
		return protocolTCPHeader{}, nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'", length, tcpAduMaxSize)
	}

	head := protocolTCPHeader{
		tid,
		tcpProtocolIdentifier,
		uint16(2 + len(pdu.Data)), // Length = sizeof(SlaveId) + sizeof(FuncCode) + Data
		slaveID,
	}

	// fill adu buffer
	adu := this.adu[0:length]
	binary.BigEndian.PutUint16(adu, head.transactionID)  // MBAP Transaction identifier
	binary.BigEndian.PutUint16(adu[2:], head.protocolID) // MBAP Protocol identifier
	binary.BigEndian.PutUint16(adu[4:], head.length)     // MBAP Length
	adu[6] = head.slaveID                                // MBAP Unit identifier
	adu[tcpHeaderMbapSize] = pdu.FuncCode                // PDU funcCode
	copy(adu[tcpHeaderMbapSize+1:], pdu.Data)            // PDU data
	return head, adu, nil
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
func decodeTCPFrame(adu []byte) (protocolTCPHeader, []byte, error) {
	if len(adu) < tcpAduMinSize { // Minimum size (including MBAP, funcCode)
		return protocolTCPHeader{}, nil, fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'", len(adu), tcpAduMinSize)
	}
	// Read length value in the header
	head := protocolTCPHeader{
		transactionID: binary.BigEndian.Uint16(adu),
		protocolID:    binary.BigEndian.Uint16(adu[2:]),
		length:        binary.BigEndian.Uint16(adu[4:]),
		slaveID:       adu[6],
	}

	pduLength := len(adu) - tcpHeaderMbapSize
	if pduLength != int(head.length-1) {
		return head, nil, fmt.Errorf("modbus: length in response '%v' does not match pdu data length '%v'",
			head.length-1, pduLength)

	}
	// The first byte after header is function code
	return head, adu[tcpHeaderMbapSize:], nil
}

// verify confirms valid data
func verifyTCPFrame(reqHead, rspHead protocolTCPHeader, reqPDU, rspPDU ProtocolDataUnit) error {
	switch {
	case rspHead.transactionID != reqHead.transactionID:
		// Check transaction ID
		return fmt.Errorf("modbus: response transaction id '%v' does not match request '%v'",
			rspHead.transactionID, reqHead.transactionID)
	case rspHead.protocolID != reqHead.protocolID:
		// Check protocol ID
		return fmt.Errorf("modbus: response protocol id '%v' does not match request '%v'",
			rspHead.protocolID, reqHead.protocolID)
	case rspHead.slaveID != reqHead.slaveID:
		// Check slaveID same
		return fmt.Errorf("modbus: response unit id '%v' does not match request '%v'",
			rspHead.slaveID, reqHead.slaveID)
	case rspPDU.FuncCode != reqPDU.FuncCode:
		// Check correct function code returned (exception)
		return responseError(rspPDU)
	case rspPDU.Data == nil || len(rspPDU.Data) == 0:
		// check Empty response
		return fmt.Errorf("modbus: response data is empty")
	}
	return nil
}

// Send the request to tcp and get the response
func (this *TCPClientProvider) Send(slaveID byte, request ProtocolDataUnit) (ProtocolDataUnit, error) {
	var response ProtocolDataUnit

	frame := this.pool.get()
	defer this.pool.put(frame)
	// add transaction id
	tid := uint16(atomic.AddUint32(&this.transactionID, 1))

	head, aduRequest, err := frame.encodeTCPFrame(tid, slaveID, request)
	if err != nil {
		return response, err
	}
	aduResponse, err := this.SendRawFrame(aduRequest)
	if err != nil {
		return response, err
	}
	rspHead, pdu, err := decodeTCPFrame(aduResponse)
	if err != nil {
		return response, err
	}
	response = ProtocolDataUnit{pdu[0], pdu[1:]}
	if err = verifyTCPFrame(head, rspHead, request, response); err != nil {
		return response, err
	}
	return response, nil
}

// SendPdu send pdu request to the remote server
func (this *TCPClientProvider) SendPdu(slaveID byte, pduRequest []byte) ([]byte, error) {
	if len(pduRequest) < pduMinSize || len(pduRequest) > pduMaxSize {
		return nil, fmt.Errorf("modbus: rspPdu size '%v' must not be between '%v' and '%v'",
			len(pduRequest), pduMinSize, pduMaxSize)
	}

	frame := this.pool.get()
	defer this.pool.put(frame)
	// add transaction id
	tid := uint16(atomic.AddUint32(&this.transactionID, 1))

	request := ProtocolDataUnit{pduRequest[0], pduRequest[1:]}
	head, aduRequest, err := frame.encodeTCPFrame(tid, slaveID, request)
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
	if err = verifyTCPFrame(head, rspHead, request, ProtocolDataUnit{rspPdu[0], rspPdu[1:]}); err != nil {
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
	this.Debug("sending [% x]", aduRequest)
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
	var cnt int
	var mErr error
	for {
		if this.Timeout > 0 {
			timeout = time.Now().Add(this.Timeout)
		}
		if err = this.conn.SetDeadline(timeout); err != nil {
			return nil, err
		}

		if cnt, err = io.ReadFull(this.conn, data[:tcpHeaderMbapSize]); err == nil {
			break
		}
		if this.autoReconnect == 0 {
			return
		}
		mErr = err
		if e, ok := err.(net.Error); ok && !e.Temporary() ||
			err != io.EOF && err != io.ErrClosedPipe ||
			strings.Contains(err.Error(), "use of closed network connection") ||
			cnt == 0 && err == io.EOF {
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
		if tryCnt++; tryCnt >= this.autoReconnect {
			err = mErr
			return
		}
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

	if this.Timeout > 0 {
		timeout = time.Now().Add(this.Timeout)
	}
	if err = this.conn.SetDeadline(timeout); err != nil {
		return nil, err
	}

	// Skip unit id
	length += tcpHeaderMbapSize - 1
	if _, err = io.ReadFull(this.conn, data[tcpHeaderMbapSize:length]); err != nil {
		return
	}
	aduResponse = data[:length]
	this.Debug("received [% x]", aduResponse)
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
