package modbus

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/thinkgos/library/elog"
)

const (
	asciiStart = ":"
	asciiEnd   = "\r\n"

	hexTable = "0123456789ABCDEF"
)

// ASCIIClientProvider implements ClientProvider interface.
type ASCIIClientProvider struct {
	serialPort
	logs
	// 请求池,所有ascii客户端共用一个请求池
	pool *sync.Pool
}

// check ASCIIClientProvider implements underlying method
var _ ClientProvider = (*ASCIIClientProvider)(nil)

// 请求池,所有ascii客户端共用一个请求池
var asciiPool = &sync.Pool{New: func() interface{} { return &protocolASCIIFrame{} }}

// NewASCIIClientProvider allocates and initializes a ASCIIClientProvider.
func NewASCIIClientProvider(address string) *ASCIIClientProvider {
	p := &ASCIIClientProvider{
		logs: logs{
			Elog: elog.NewElog(nil),
		},
		pool: asciiPool,
	}
	p.Address = address
	p.Timeout = SerialDefaultTimeout
	p.autoReconnect = SerialDefaultAutoReconnect
	return p
}

// encode slaveID & PDU to a ASCII frame,return adu
//  Start           : 1 char
//  slaveID         : 2 chars
//  ---- data Unit ----
//  Function        : 2 chars
//  Data            : 0 up to 2x252 chars
//  ---- checksun ----
//  LRC             : 2 chars
//  End             : 2 chars
func (this *protocolASCIIFrame) encode(slaveID byte, pdu *ProtocolDataUnit) ([]byte, error) {
	length := len(pdu.Data) + 3
	if length > asciiAduMaxSize {
		return nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'", length, asciiAduMaxSize)
	}
	// save
	this.slaveID = slaveID
	this.pdu.FuncCode = pdu.FuncCode

	// Exclude the beginning colon and terminating CRLF pair characters
	var lrc lrc
	lrc.reset().push(this.slaveID)
	lrc.push(pdu.FuncCode).push(pdu.Data...)
	lrcVal := lrc.value()

	// real ascii frame to send,
	// includeing asciiStart + ( slaveID + funciton + data + lrc ) + CRLF
	frame := this.adu[: 0 : (len(pdu.Data)+3)*2+3]
	frame = append(frame, []byte(asciiStart)...) // the beginning colon characters
	// the real adu
	frame = append(frame, hexTable[this.slaveID>>4], hexTable[this.slaveID&0x0F]) // slave ID
	frame = append(frame, hexTable[pdu.FuncCode>>4], hexTable[pdu.FuncCode&0x0F]) // pdu funcCode
	for _, v := range pdu.Data {
		frame = append(frame, hexTable[v>>4], hexTable[v&0x0F]) // pdu data
	}
	frame = append(frame, hexTable[lrcVal>>4], hexTable[lrcVal&0x0F]) // lrc
	// terminating CRLF characters
	return append(frame, []byte(asciiEnd)...), nil
}

// decode extracts slaveID & PDU from ASCII frame and verify LRC.
func (this *protocolASCIIFrame) decode(adu []byte) (uint8, *ProtocolDataUnit, []byte, error) {
	if len(adu) < asciiAduMinSize+6 { // Minimum size (including address, function and LRC)
		return 0, nil, nil, fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'", len(adu), 9)
	}
	switch {
	case len(adu)%2 != 1:
		// Length excluding colon must be an even number
		return 0, nil, nil, fmt.Errorf("modbus: response length '%v' is not an even number", len(adu)-1)
	case string(adu[0:len(asciiStart)]) != asciiStart:
		// First char must be a colons
		return 0, nil, nil, fmt.Errorf("modbus: response frame '%x'... is not started with '%x'",
			string(adu[0:len(asciiStart)]), asciiStart)
	case string(adu[len(adu)-len(asciiEnd):]) != asciiEnd:
		// 2 last chars must be \r\n
		return 0, nil, nil, fmt.Errorf("modbus: response frame ...'%x' is not ended with '%x'",
			string(adu[len(adu)-len(asciiEnd):]), asciiEnd)
	}

	// real adu  pass Start and CRLF
	dat := adu[1 : len(adu)-2]
	buf := make([]byte, hex.DecodedLen(len(dat)))
	length, err := hex.Decode(buf, dat)
	if err != nil {
		return 0, nil, nil, err
	}
	// Calculate checksum
	var lrc lrc
	sum := lrc.reset().push(buf[:length-1]...).value()
	if buf[length-1] != sum { // LRC
		return 0, nil, nil, fmt.Errorf("modbus: response lrc '%x' does not match expected '%x'", buf[length-1], sum)
	}
	return buf[0], &ProtocolDataUnit{buf[1], buf[2 : length-1]}, buf[1 : length-1], nil
}

// verify confirms vaild data
func (this *protocolASCIIFrame) verify(reqSlaveID, rspSlaveID uint8, reqPDU, rspPDU *ProtocolDataUnit) error {
	return verify(reqSlaveID, rspSlaveID, reqPDU, rspPDU)
}

// Send request to the remote server,it implements on SendRawFrame
func (this *ASCIIClientProvider) Send(slaveID byte, request *ProtocolDataUnit) (*ProtocolDataUnit, error) {
	frame := this.pool.Get().(*protocolASCIIFrame)
	defer this.pool.Put(frame)
	aduRequest, err := frame.encode(slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := this.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspSlaveID, response, _, err := frame.decode(aduResponse)
	if err != nil {
		return nil, err
	}
	if err = frame.verify(slaveID, rspSlaveID, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// SendPdu send pdu request to the remote server
func (this *ASCIIClientProvider) SendPdu(slaveID byte, pduRequest []byte) (pduResponse []byte, err error) {
	if len(pduRequest) < pduMinSize || len(pduRequest) > pduMaxSize {
		return nil, fmt.Errorf("modbus: pdu size '%v' must not be between '%v' and '%v'",
			len(pduRequest), pduMinSize, pduMaxSize)
	}

	frame := this.pool.Get().(*protocolASCIIFrame)
	defer this.pool.Put(frame)
	request := &ProtocolDataUnit{pduRequest[0], pduRequest[1:]}
	aduRequest, err := frame.encode(slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := this.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspSlaveID, response, pdu, err := frame.decode(aduResponse)
	if err != nil {
		return nil, err
	}
	if err = frame.verify(slaveID, rspSlaveID, request, response); err != nil {
		return nil, err
	}
	return pdu, nil
}

// SendRawFrame send Adu frame
func (this *ASCIIClientProvider) SendRawFrame(aduRequest []byte) (aduResponse []byte, err error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	// check  port is connected
	if !this.isConnected() {
		return nil, fmt.Errorf("modbus: Client is not connected")
	}

	// Send the request
	this.Debug("modbus: sending % x", aduRequest)
	var tryCnt byte
	for {
		_, err = this.port.Write(aduRequest)
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

	// Get the response
	var n int
	var data [asciiCharacterMaxSize]byte
	length := 0
	for {
		if n, err = this.port.Read(data[length:]); err != nil {
			return
		}
		length += n
		if length >= asciiCharacterMaxSize || n == 0 {
			break
		}
		// Expect end of frame in the data received
		if length > asciiAduMinSize {
			if string(data[length-len(asciiEnd):length]) == asciiEnd {
				break
			}
		}
	}
	aduResponse = data[:length]
	this.Debug("modbus: received % x\n", aduResponse)
	return
}
