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

	"github.com/goburrow/serial"
)

const (
	// TCPDefaultTimeout TCP Default timeout
	TCPDefaultTimeout = 1 * time.Second
	// TCPDefaultAutoReconnect TCP Default auto reconnect count
	TCPDefaultAutoReconnect = 1
)

// TCPClientProvider implements ClientProvider interface.
type TCPClientProvider struct {
	logger
	address string
	mu      sync.Mutex
	// TCP connection
	conn net.Conn
	// Connect & Read timeout
	timeout time.Duration
	// For synchronization between messages of server & client
	transactionID uint32
	// request
	*pool
}

// check TCPClientProvider implements the interface ClientProvider underlying method
var _ ClientProvider = (*TCPClientProvider)(nil)

// request pool, all TCP client use this pool
var tcpPool = newPool(tcpAduMaxSize)

// NewTCPClientProvider allocates a new TCPClientProvider.
func NewTCPClientProvider(address string, opts ...ClientProviderOption) *TCPClientProvider {
	p := &TCPClientProvider{
		address: address,
		timeout: TCPDefaultTimeout,
		pool:    tcpPool,
		logger:  newLogger("modbusTCPMaster =>"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// encode modbus application protocol header & pdu to TCP frame,return adu.
//  ---- MBAP header ----
//  Transaction identifier: 2 bytes
//  Protocol identifier: 2 bytes
//  Length: 2 bytes
//  Unit identifier: 1 byte
//  ---- data Unit ----
//  Function code: 1 byte
//  Data: n bytes
func (sf *protocolFrame) encodeTCPFrame(tid uint16, slaveID byte,
	pdu ProtocolDataUnit) (protocolTCPHeader, []byte, error) {
	length := tcpHeaderMbapSize + 1 + len(pdu.Data)
	if length > tcpAduMaxSize {
		return protocolTCPHeader{}, nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'",
			length, tcpAduMaxSize)
	}

	head := protocolTCPHeader{
		tid,
		tcpProtocolIdentifier,
		uint16(2 + len(pdu.Data)), // sizeof(SlaveId) + sizeof(FuncCode) + Data
		slaveID,
	}

	// fill adu buffer
	adu := sf.adu[0:length]
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
		return protocolTCPHeader{}, nil, fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'",
			len(adu), tcpAduMinSize)
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
	case rspHead.transactionID != reqHead.transactionID: // Check transaction ID
		return fmt.Errorf("modbus: response transaction id '%v' does not match request '%v'",
			rspHead.transactionID, reqHead.transactionID)

	case rspHead.protocolID != reqHead.protocolID: // Check protocol ID
		return fmt.Errorf("modbus: response protocol id '%v' does not match request '%v'",
			rspHead.protocolID, reqHead.protocolID)

	case rspHead.slaveID != reqHead.slaveID: // Check slaveID same
		return fmt.Errorf("modbus: response unit id '%v' does not match request '%v'",
			rspHead.slaveID, reqHead.slaveID)

	case rspPDU.FuncCode != reqPDU.FuncCode: // Check correct function code returned (exception)
		return responseError(rspPDU)

	case rspPDU.Data == nil || len(rspPDU.Data) == 0: // check Empty response
		return fmt.Errorf("modbus: response data is empty")
	}
	return nil
}

// Send the request to tcp and get the response
func (sf *TCPClientProvider) Send(slaveID byte, request ProtocolDataUnit) (ProtocolDataUnit, error) {
	var response ProtocolDataUnit

	frame := sf.pool.get()
	defer sf.pool.put(frame)
	// add transaction id
	tid := uint16(atomic.AddUint32(&sf.transactionID, 1))

	head, aduRequest, err := frame.encodeTCPFrame(tid, slaveID, request)
	if err != nil {
		return response, err
	}
	aduResponse, err := sf.SendRawFrame(aduRequest)
	if err != nil {
		return response, err
	}
	rspHead, pdu, err := decodeTCPFrame(aduResponse)
	if err != nil {
		return response, err
	}
	response = ProtocolDataUnit{pdu[0], pdu[1:]}
	err = verifyTCPFrame(head, rspHead, request, response)
	return response, err
}

// SendPdu send pdu request to the remote server
func (sf *TCPClientProvider) SendPdu(slaveID byte, pduRequest []byte) ([]byte, error) {
	if len(pduRequest) < pduMinSize || len(pduRequest) > pduMaxSize {
		return nil, fmt.Errorf("modbus: rspPdu size '%v' must not be between '%v' and '%v'",
			len(pduRequest), pduMinSize, pduMaxSize)
	}

	frame := sf.pool.get()
	defer sf.pool.put(frame)
	// add transaction id
	tid := uint16(atomic.AddUint32(&sf.transactionID, 1))

	request := ProtocolDataUnit{pduRequest[0], pduRequest[1:]}
	head, aduRequest, err := frame.encodeTCPFrame(tid, slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := sf.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspHead, rspPdu, err := decodeTCPFrame(aduResponse)
	if err != nil {
		return nil, err
	}
	response := ProtocolDataUnit{rspPdu[0], rspPdu[1:]}
	if err = verifyTCPFrame(head, rspHead, request, response); err != nil {
		return nil, err
	}
	// rspPdu pass tcpMBAP head
	return rspPdu, nil
}

// SendRawFrame send raw adu request frame
func (sf *TCPClientProvider) SendRawFrame(aduRequest []byte) (aduResponse []byte, err error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if err = sf.connect(); err != nil {
		return
	}
	// Send data
	sf.Debugf("sending [% x]", aduRequest)
	// Set write and read timeout
	var timeout time.Time

	if sf.timeout > 0 {
		timeout = time.Now().Add(sf.timeout)
	}
	if err = sf.conn.SetDeadline(timeout); err != nil {
		return nil, err
	}

	if _, err = sf.conn.Write(aduRequest); err != nil {
		return
	}

	// Read header first
	var data [tcpAduMaxSize]byte
	var cnt int

	if sf.timeout > 0 {
		timeout = time.Now().Add(sf.timeout)
	}
	if err = sf.conn.SetDeadline(timeout); err != nil {
		return nil, err
	}

	if cnt, err = io.ReadFull(sf.conn, data[:tcpHeaderMbapSize]); err != nil {
		if e, ok := err.(net.Error); (ok && !e.Temporary() && !e.Timeout()) ||
			(err != io.EOF && err == io.ErrClosedPipe) ||
			strings.Contains(err.Error(), "use of closed network connection") ||
			(cnt == 0 && err == io.EOF) {
			sf.close()
		}
		return
	}

	// Read length, ignore transaction & protocol id (4 bytes)
	length := int(binary.BigEndian.Uint16(data[4:]))
	switch {
	case length <= 0:
		_ = sf.flush(data[:])
		err = fmt.Errorf("modbus: length in response header '%v' must not be zero", length)
		return
	case length > (tcpAduMaxSize - (tcpHeaderMbapSize - 1)):
		_ = sf.flush(data[:])
		err = fmt.Errorf("modbus: length in response header '%v' must not greater than '%v'",
			length, tcpAduMaxSize-tcpHeaderMbapSize+1)
		return
	}

	if sf.timeout > 0 {
		timeout = time.Now().Add(sf.timeout)
	}
	if err = sf.conn.SetDeadline(timeout); err != nil {
		return nil, err
	}

	// Skip unit id
	length += tcpHeaderMbapSize - 1
	if _, err = io.ReadFull(sf.conn, data[tcpHeaderMbapSize:length]); err != nil {
		return
	}
	aduResponse = data[:length]
	sf.Debugf("received [% x]", aduResponse)
	return aduResponse, nil
}

// Connect establishes a new connection to the address in Address.
// Connect and Close are exported so that multiple requests can be done with one session
func (sf *TCPClientProvider) Connect() error {
	sf.mu.Lock()
	err := sf.connect()
	sf.mu.Unlock()
	return err
}

// Caller must hold the mutex before calling this method.
func (sf *TCPClientProvider) connect() error {
	if sf.conn == nil {
		dialer := &net.Dialer{Timeout: sf.timeout}
		conn, err := dialer.Dial("tcp", sf.address)
		if err != nil {
			return err
		}
		sf.conn = conn
	}
	return nil
}

// IsConnected returns a bool signifying whether
// the client is connected or not.
func (sf *TCPClientProvider) IsConnected() bool {
	sf.mu.Lock()
	b := sf.conn != nil
	sf.mu.Unlock()
	return b
}

// Caller must hold the mutex before calling this method.
func (sf *TCPClientProvider) close() (err error) {
	if sf.conn != nil {
		err = sf.conn.Close()
		sf.conn = nil
	}
	return
}

// Close closes current connection.
func (sf *TCPClientProvider) Close() (err error) {
	sf.mu.Lock()
	err = sf.close()
	sf.mu.Unlock()
	return
}

func (sf *TCPClientProvider) setSerialConfig(serial.Config) {}

func (sf *TCPClientProvider) setTCPTimeout(t time.Duration) {
	sf.timeout = t
}

// flush flushes pending data in the connection,
// returns io.EOF if connection is closed.
func (sf *TCPClientProvider) flush(b []byte) (err error) {
	if err = sf.conn.SetReadDeadline(time.Now()); err != nil {
		return
	}
	// timeout setting will be reset when reading
	if _, err = sf.conn.Read(b); err != nil {
		// Ignore timeout error
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			err = nil
		}
	}
	return
}
