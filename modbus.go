/*!
 * Constants which defines the format of a modbus frame. The example is
 * shown for a Modbus RTU/ASCII frame. Note that the Modbus PDU is not
 * dependent on the underlying transport.
 *
 * <code>
 * <------------------------ MODBUS SERIAL LINE ADU (1) ------------------->
 *              <----------- MODBUS PDU (1') ---------------->
 *  +-----------+---------------+----------------------------+-------------+
 *  | Address   | Function Code | Data                       | CRC/LRC     |
 *  +-----------+---------------+----------------------------+-------------+
 *  |           |               |                                   |
 * (2)        (3/2')           (3')                                (4)
 *
 * (1)  ... SerADUMaxSize    = 256
 * (2)  ... SerAddressOffset = 0
 * (3)  ... SerPDUOffset     = 1
 * (4)  ... SerCrcSize       = 2
 *      ... SerLrcSize       = 1
 *
 * (1') ... SerPDUMaxSize         = 253
 * (2') ... SerPDUFuncCodeOffset  = 0
 * (3') ... SerPDUDataOffset       = 1
 * </code>
 */

/*!
 * <------------------------ MODBUS TCP/IP ADU(1) ------------------------->
 *                              <----------- MODBUS PDU (1') -------------->
 *  +-----------+---------------+------------------------------------------+
 *  | TID | PID | Length | UID  | Function Code  | Data                    |
 *  +-----------+---------------+------------------------------------------+
 *  |     |     |        |      |
 * (2)   (3)   (4)      (5)    (6)
 *
 * (2)  ... TCPTidOffset    = 0 (Transaction Identifier - 2 Byte)
 * (3)  ... TCPPidOffset    = 2 (Protocol Identifier - 2 Byte)
 * (4)  ... TCPLengthOffset = 4 (Number of bytes - 2 Byte)( UID + PDU length )
 * (5)  ... TCPUidOffset    = 6 (Unit Identifier - 1 Byte)
 * (6)  ... TCPPDUOffset    = 7 (Modbus PDU )
 *
 * (1)  ... TCPADUMaxSize   = 260 Modbus TCP/IP Application Data Unit
 * (1') ... SerPDUMaxSize   = 253 Modbus Protocol Data Unit
 */

/*
Package modbus provides a client for modbus TCP and RTU/ASCII.contain modbus TCP server
*/
package modbus

import (
	"fmt"
	"time"

	"github.com/goburrow/serial"
)

// proto address limit.
const (
	AddressBroadCast = 0
	AddressMin       = 1
	AddressMax       = 247
)

const (
	pduMinSize = 1   // funcCode(1)
	pduMaxSize = 253 // funcCode(1) + data(252)

	rtuAduMinSize = 4   // address(1) + funcCode(1) + crc(2)
	rtuAduMaxSize = 256 // address(1) + PDU(253) + crc(2)

	asciiAduMinSize       = 3
	asciiAduMaxSize       = 256
	asciiCharacterMaxSize = 513

	tcpProtocolIdentifier = 0x0000
	// Modbus Application Protocol
	tcpHeaderMbapSize = 7 // MBAP header
	tcpAduMinSize     = 8 // MBAP + funcCode
	tcpAduMaxSize     = 260
)

// proto register limit
const (
	// Bits
	ReadBitsQuantityMin  = 1    // 0x0001
	ReadBitsQuantityMax  = 2000 // 0x07d0
	WriteBitsQuantityMin = 1    // 1
	WriteBitsQuantityMax = 1968 // 0x07b0
	// 16 Bits
	ReadRegQuantityMin             = 1   // 1
	ReadRegQuantityMax             = 125 // 0x007d
	WriteRegQuantityMin            = 1   // 1
	WriteRegQuantityMax            = 123 // 0x007b
	ReadWriteOnReadRegQuantityMin  = 1   // 1
	ReadWriteOnReadRegQuantityMax  = 125 // 0x007d
	ReadWriteOnWriteRegQuantityMin = 1   // 1
	ReadWriteOnWriteRegQuantityMax = 121 // 0x0079
)

// Function Code
const (
	// Bit access
	FuncCodeReadDiscreteInputs = 2
	FuncCodeReadCoils          = 1
	FuncCodeWriteSingleCoil    = 5
	FuncCodeWriteMultipleCoils = 15

	// 16-bit access
	FuncCodeReadInputRegisters         = 4
	FuncCodeReadHoldingRegisters       = 3
	FuncCodeWriteSingleRegister        = 6
	FuncCodeWriteMultipleRegisters     = 16
	FuncCodeReadWriteMultipleRegisters = 23
	FuncCodeMaskWriteRegister          = 22
	FuncCodeReadFIFOQueue              = 24
	FuncCodeOtherReportSlaveID         = 17
	// FuncCodeDiagReadException          = 7
	// FuncCodeDiagDiagnostic             = 8
	// FuncCodeDiagGetComEventCnt         = 11
	// FuncCodeDiagGetComEventLog         = 12
)

// Exception Code
const (
	ExceptionCodeIllegalFunction                    = 1
	ExceptionCodeIllegalDataAddress                 = 2
	ExceptionCodeIllegalDataValue                   = 3
	ExceptionCodeServerDeviceFailure                = 4
	ExceptionCodeAcknowledge                        = 5
	ExceptionCodeServerDeviceBusy                   = 6
	ExceptionCodeNegativeAcknowledge                = 7
	ExceptionCodeMemoryParityError                  = 8
	ExceptionCodeGatewayPathUnavailable             = 10
	ExceptionCodeGatewayTargetDeviceFailedToRespond = 11
)

// ExceptionError implements error interface.
type ExceptionError struct {
	ExceptionCode byte
}

// Error converts known modbus exception code to error message.
func (e *ExceptionError) Error() string {
	var name string
	switch e.ExceptionCode {
	case ExceptionCodeIllegalFunction:
		name = "illegal function"
	case ExceptionCodeIllegalDataAddress:
		name = "illegal data address"
	case ExceptionCodeIllegalDataValue:
		name = "illegal data value"
	case ExceptionCodeServerDeviceFailure:
		name = "server device failure"
	case ExceptionCodeAcknowledge:
		name = "acknowledge"
	case ExceptionCodeServerDeviceBusy:
		name = "server device busy"
	case ExceptionCodeNegativeAcknowledge:
		name = "Negative Acknowledge"
	case ExceptionCodeMemoryParityError:
		name = "memory parity error"
	case ExceptionCodeGatewayPathUnavailable:
		name = "gateway path unavailable"
	case ExceptionCodeGatewayTargetDeviceFailedToRespond:
		name = "gateway target device failed to respond"
	default:
		name = "unknown"
	}
	return fmt.Sprintf("modbus: exception '%v' (%s)", e.ExceptionCode, name)
}

// protocolTCPHeader independent of underlying communication layers.
type protocolTCPHeader struct {
	transactionID uint16
	protocolID    uint16
	length        uint16
	slaveID       uint8 // only modbus RTU and ascii
}

// ProtocolDataUnit (PDU) is independent of underlying communication layers.
type ProtocolDataUnit struct {
	FuncCode byte
	Data     []byte
}

// protocolFrame protocol frame in pool
type protocolFrame struct {
	adu []byte
}

// ClientProvider is the interface implements underlying methods.
type ClientProvider interface {
	// Connect try to connect the remote server
	Connect() error
	// IsConnected returns a bool signifying whether
	// the client is connected or not.
	IsConnected() bool
	// LogMode set enable or diable log output when you has set logger
	LogMode(enable bool)
	// Close disconnect the remote server
	Close() error
	// Send request to the remote server,it implements on SendRawFrame
	Send(slaveID byte, request ProtocolDataUnit) (ProtocolDataUnit, error)
	// SendPdu send pdu request to the remote server
	SendPdu(slaveID byte, pduRequest []byte) (pduResponse []byte, err error)
	// SendRawFrame send raw frame to the remote server
	SendRawFrame(aduRequest []byte) (aduResponse []byte, err error)

	// private interface
	// setLogProvider set logger provider
	setLogProvider(p LogProvider)
	// setSerialConfig set serial config
	setSerialConfig(config serial.Config)
	// setTCPTimeout set tcp connect & read timeout
	setTCPTimeout(t time.Duration)
}

// LogProvider RFC5424 log message levels only Debug and Error
type LogProvider interface {
	Errorf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}
