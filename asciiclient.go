package modbus

import (
	"encoding/hex"
	"fmt"
)

// protocol frame: asciiStart + ( slaveID + functionCode + data + lrc ) + CR + LF.
const (
	asciiStart = ":"
	asciiEnd   = "\r\n"
	hexTable   = "0123456789ABCDEF"
)

// ASCIIClientProvider implements ClientProvider interface.
type ASCIIClientProvider struct {
	serialPort
	logger
	*pool
}

// check ASCIIClientProvider implements the interface ClientProvider underlying method.
var _ ClientProvider = (*ASCIIClientProvider)(nil)

// request pool, all ASCII client use this pool.
var asciiPool = newPool(asciiCharacterMaxSize)

// NewASCIIClientProvider allocates and initializes a ASCIIClientProvider.
// it will use default /dev/ttyS0 19200 8 1 N and timeout 1000.
func NewASCIIClientProvider(opts ...ClientProviderOption) *ASCIIClientProvider {
	p := &ASCIIClientProvider{
		logger: newLogger("modbusASCIIMaster => "),
		pool:   asciiPool,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// encode slaveID & PDU to a ASCII frame,return adu
//  Start           : 1 char
//  slaveID         : 2 chars
//  ---- data Unit ----
//  Function        : 2 chars
//  Data            : 0 up to 2x252 chars
//  ---- checksum ----
//  LRC             : 2 chars
//  End             : 2 chars
func (sf *protocolFrame) encodeASCIIFrame(slaveID byte, pdu ProtocolDataUnit) ([]byte, error) {
	length := len(pdu.Data) + 3
	if length > asciiAduMaxSize {
		return nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'", length, asciiAduMaxSize)
	}

	// Exclude the beginning colon and terminating CRLF pair characters
	lrcVal := new(LRC).
		Reset().
		Push(slaveID).Push(pdu.FuncCode).Push(pdu.Data...).
		Value()

	// real ascii frame to send,
	// including asciiStart + ( slaveID + functionCode + data + lrc ) + CRLF
	frame := sf.adu[: 0 : length*2+3]
	frame = append(frame, []byte(asciiStart)...) // the beginning colon characters
	// the real adu
	frame = append(frame,
		hexTable[slaveID>>4], hexTable[slaveID&0x0f], // slave ID
		hexTable[pdu.FuncCode>>4], hexTable[pdu.FuncCode&0x0f]) // pdu funcCode
	for _, v := range pdu.Data {
		frame = append(frame, hexTable[v>>4], hexTable[v&0x0f]) // pdu data
	}
	frame = append(frame, hexTable[lrcVal>>4], hexTable[lrcVal&0x0f]) // lrc value
	// terminating CRLF characters
	return append(frame, []byte(asciiEnd)...), nil
}

// decode extracts slaveID & PDU from ASCII frame and verify LRC.
func decodeASCIIFrame(adu []byte) (uint8, []byte, error) {
	if len(adu) < asciiAduMinSize+6 { // Minimum size (including address, function and LRC)
		return 0, nil, fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'", len(adu), 9)
	}
	switch {
	case len(adu)%2 != 1: // Length excluding colon must be an even number
		return 0, nil, fmt.Errorf("modbus: response length '%v' is not an even number", len(adu)-1)
	case string(adu[0:len(asciiStart)]) != asciiStart: // First char must be a colons
		return 0, nil, fmt.Errorf("modbus: response frame '%x'... is not started with '%x'",
			string(adu[0:len(asciiStart)]), asciiStart)
	case string(adu[len(adu)-len(asciiEnd):]) != asciiEnd: // 2 last chars must be \r\n
		return 0, nil, fmt.Errorf("modbus: response frame ...'%x' is not ended with '%x'",
			string(adu[len(adu)-len(asciiEnd):]), asciiEnd)
	}

	// real adu  pass Start and CRLF
	dat := adu[1 : len(adu)-2]
	buf := make([]byte, hex.DecodedLen(len(dat)))
	length, err := hex.Decode(buf, dat)
	if err != nil {
		return 0, nil, err
	}
	// Calculate checksum
	lrcVal := new(LRC).Reset().Push(buf[:length-1]...).Value()
	if buf[length-1] != lrcVal { // LRC
		return 0, nil, fmt.Errorf("modbus: response lrc '%x' does not match expected '%x'", buf[length-1], lrcVal)
	}
	return buf[0], buf[1 : length-1], nil
}

// Send request to the remote server,it implements on SendRawFrame.
func (sf *ASCIIClientProvider) Send(slaveID byte, request ProtocolDataUnit) (ProtocolDataUnit, error) {
	var response ProtocolDataUnit

	frame := sf.pool.get()
	defer sf.pool.put(frame)

	aduRequest, err := frame.encodeASCIIFrame(slaveID, request)
	if err != nil {
		return response, err
	}
	aduResponse, err := sf.SendRawFrame(aduRequest)
	if err != nil {
		return response, err
	}
	rspSlaveID, pdu, err := decodeASCIIFrame(aduResponse)
	if err != nil {
		return response, err
	}

	response = ProtocolDataUnit{pdu[0], pdu[1:]}
	if err = verify(slaveID, rspSlaveID, request, response); err != nil {
		return response, err
	}
	return response, nil
}

// SendPdu send pdu request to the remote server.
func (sf *ASCIIClientProvider) SendPdu(slaveID byte, pduRequest []byte) ([]byte, error) {
	if len(pduRequest) < pduMinSize || len(pduRequest) > pduMaxSize {
		return nil, fmt.Errorf("modbus: pdu size '%v' must not be between '%v' and '%v'",
			len(pduRequest), pduMinSize, pduMaxSize)
	}

	frame := sf.pool.get()
	defer sf.pool.put(frame)

	request := ProtocolDataUnit{pduRequest[0], pduRequest[1:]}
	aduRequest, err := frame.encodeASCIIFrame(slaveID, request)
	if err != nil {
		return nil, err
	}
	aduResponse, err := sf.SendRawFrame(aduRequest)
	if err != nil {
		return nil, err
	}
	rspSlaveID, pdu, err := decodeASCIIFrame(aduResponse)
	if err != nil {
		return nil, err
	}
	response := ProtocolDataUnit{pdu[0], pdu[1:]}
	if err = verify(slaveID, rspSlaveID, request, response); err != nil {
		return nil, err
	}
	return pdu, nil
}

// SendRawFrame send Adu frame.
func (sf *ASCIIClientProvider) SendRawFrame(aduRequest []byte) (aduResponse []byte, err error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if err = sf.connect(); err != nil {
		return nil, err
	}
	// Send the request
	sf.Debugf("sending [% x]", aduRequest)

	_, err = sf.port.Write(aduRequest)
	if err == nil {
		sf.close()
		return
	}

	// Get the response
	var n int
	var data [asciiCharacterMaxSize]byte
	length := 0
	for {
		if n, err = sf.port.Read(data[length:]); err != nil {
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
	sf.Debugf("received [% x]", aduResponse)
	return aduResponse, nil
}
