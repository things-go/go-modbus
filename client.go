package modbus

import (
	"encoding/binary"
	"fmt"
)

// check implements Client interface
var _ Client = (*client)(nil)

// client implements Client interface
type client struct {
	ClientProvider
}

// NewClient creates a new modbus client with given backend handler.
func NewClient(p ClientProvider) Client {
	return &client{p}
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x01)
//  Starting address      : 2 bytes
//  Quantity of coils     : 2 bytes
// Response:
//  Function code         : 1 byte (0x01)
//  Byte count            : 1 byte
//  Coil status           : N* bytes (=N or N+1)
//  return coils status
func (sf *client) ReadCoils(slaveID byte, address, quantity uint16) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	if quantity < ReadBitsQuantityMin || quantity > ReadBitsQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadBitsQuantityMin, ReadBitsQuantityMax)

	}

	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCodeReadCoils,
		pduDataBlock(address, quantity),
	})

	switch {
	case err != nil:
		return nil, err
	case len(response.Data)-1 != int(response.Data[0]):
		return nil, fmt.Errorf("modbus: response byte size '%v' does not match count '%v'",
			len(response.Data)-1, int(response.Data[0]))
	case uint16(response.Data[0]) != (quantity+7)/8:
		return nil, fmt.Errorf("modbus: response byte size '%v' does not match quantity to bytes '%v'",
			response.Data[0], (quantity+7)/8)
	}
	return response.Data[1:], nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x02)
//  Starting address      : 2 bytes
//  Quantity of inputs    : 2 bytes
// Response:
//  Function code         : 1 byte (0x02)
//  Byte count            : 1 byte
//  Input status          : N* bytes (=N or N+1)
//  return result data
func (sf *client) ReadDiscreteInputs(slaveID byte, address, quantity uint16) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	if quantity < ReadBitsQuantityMin || quantity > ReadBitsQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadBitsQuantityMin, ReadBitsQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadDiscreteInputs,
		Data:     pduDataBlock(address, quantity),
	})

	switch {
	case err != nil:
		return nil, err
	case len(response.Data)-1 != int(response.Data[0]):
		return nil, fmt.Errorf("modbus: response byte size '%v' does not match count '%v'",
			len(response.Data)-1, response.Data[0])
	case uint16(response.Data[0]) != (quantity+7)/8:
		return nil, fmt.Errorf("modbus: response byte size '%v' does not match quantity to bytes '%v'",
			response.Data[0], (quantity+7)/8)
	}
	return response.Data[1:], nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x03)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x03)
//  Byte count            : 1 byte
//  Register value        : Nx2 bytes
func (sf *client) ReadHoldingRegistersBytes(slaveID byte, address, quantity uint16) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	if quantity < ReadRegQuantityMin || quantity > ReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadRegQuantityMin, ReadRegQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadHoldingRegisters,
		Data:     pduDataBlock(address, quantity),
	})

	switch {
	case err != nil:
		return nil, err
	case len(response.Data)-1 != int(response.Data[0]):
		return nil, fmt.Errorf("modbus: response data size '%v' does not match count '%v'",
			len(response.Data)-1, response.Data[0])
	case uint16(response.Data[0]) != quantity*2:
		return nil, fmt.Errorf("modbus: response data size '%v' does not match quantity to bytes '%v'",
			response.Data[0], quantity*2)
	}
	return response.Data[1:], nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x03)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x03)
//  Byte count            : 1 byte
//  Register value        : N 2-bytes
func (sf *client) ReadHoldingRegisters(slaveID byte, address, quantity uint16) ([]uint16, error) {
	b, err := sf.ReadHoldingRegistersBytes(slaveID, address, quantity)
	if err != nil {
		return nil, err
	}
	return bytes2Uint16(b), nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x04)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x04)
//  Byte count            : 1 byte
//  Input registers       : Nx2 bytes
func (sf *client) ReadInputRegistersBytes(slaveID byte, address, quantity uint16) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	if quantity < ReadRegQuantityMin || quantity > ReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadRegQuantityMin, ReadRegQuantityMax)

	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadInputRegisters,
		Data:     pduDataBlock(address, quantity),
	})

	switch {
	case err != nil:
		return nil, err
	}

	if len(response.Data)-1 != int(response.Data[0]) {
		return nil, fmt.Errorf("modbus: response data size '%v' does not match count '%v'",
			len(response.Data)-1, response.Data[0])
	}
	if uint16(response.Data[0]) != quantity*2 {
		return nil, fmt.Errorf("modbus: response data size '%v' does not match quantity to bytes '%v'",
			response.Data[0], quantity*2)
	}
	return response.Data[1:], nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x04)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x04)
//  Byte count            : 1 byte
//  Input registers       : N 2-bytes
func (sf *client) ReadInputRegisters(slaveID byte, address, quantity uint16) ([]uint16, error) {
	b, err := sf.ReadInputRegistersBytes(slaveID, address, quantity)
	if err != nil {
		return nil, err
	}
	return bytes2Uint16(b), nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
// Response:
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
func (sf *client) WriteSingleCoil(slaveID byte, address uint16, isOn bool) error {
	if slaveID > AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, AddressMax)
	}
	var value uint16
	if isOn { // The requested ON/OFF state can only be 0xFF00 and 0x0000
		value = 0xFF00
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteSingleCoil,
		Data:     pduDataBlock(address, value),
	})

	switch {
	case err != nil:
		return err
	case len(response.Data) != 4:
		// Fixed response length
		return fmt.Errorf("modbus: response data size '%v' does not match expected '%v'",
			len(response.Data), 4)
	case binary.BigEndian.Uint16(response.Data) != address:
		// check address
		return fmt.Errorf("modbus: response address '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data), address)
	case binary.BigEndian.Uint16(response.Data[2:]) != value:
		// check value
		return fmt.Errorf("modbus: response value '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), value)
	}
	return nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
// Response:
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
func (sf *client) WriteSingleRegister(slaveID byte, address, value uint16) error {
	if slaveID > AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, AddressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteSingleRegister,
		Data:     pduDataBlock(address, value),
	})

	switch {
	case err != nil:
		return err
	case len(response.Data) != 4:
		// Fixed response length
		return fmt.Errorf("modbus: response data size '%v' does not match expected '%v'",
			len(response.Data), 4)
	case binary.BigEndian.Uint16(response.Data) != address:
		return fmt.Errorf("modbus: response address '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data), address)
	case binary.BigEndian.Uint16(response.Data[2:]) != value:
		return fmt.Errorf("modbus: response value '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), value)
	}
	return nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x0F)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Outputs value         : N* bytes
// Response:
//  Function code         : 1 byte (0x0F)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
func (sf *client) WriteMultipleCoils(slaveID byte, address, quantity uint16, value []byte) error {
	if slaveID > AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, AddressMax)
	}
	if quantity < WriteBitsQuantityMin || quantity > WriteBitsQuantityMax {
		return fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, WriteBitsQuantityMin, WriteBitsQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteMultipleCoils,
		Data:     pduDataBlockSuffix(value, address, quantity),
	})

	switch {
	case err != nil:
		return err
	case len(response.Data) != 4:
		// Fixed response length
		return fmt.Errorf("modbus: response data size '%v' does not match expected '%v'",
			len(response.Data), 4)
	case binary.BigEndian.Uint16(response.Data) != address:
		return fmt.Errorf("modbus: response address '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data), address)
	case binary.BigEndian.Uint16(response.Data[2:]) != quantity:
		return fmt.Errorf("modbus: response quantity '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), quantity)
	}
	return nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Registers value       : N* bytes
// Response:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (sf *client) WriteMultipleRegisters(slaveID byte, address, quantity uint16, value []byte) error {
	if slaveID > AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, AddressMax)
	}
	if quantity < WriteRegQuantityMin || quantity > WriteRegQuantityMax {
		return fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, WriteRegQuantityMin, WriteRegQuantityMax)
	}

	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteMultipleRegisters,
		Data:     pduDataBlockSuffix(value, address, quantity),
	})

	switch {
	case err != nil:
		return err
	case len(response.Data) != 4:
		// Fixed response length
		return fmt.Errorf("modbus: response data size '%v' does not match expected '%v'",
			len(response.Data), 4)
	case binary.BigEndian.Uint16(response.Data) != address:
		return fmt.Errorf("modbus: response address '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data), address)
	case binary.BigEndian.Uint16(response.Data[2:]) != quantity:
		return fmt.Errorf("modbus: response quantity '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), quantity)
	}
	return nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x16)
//  Reference address     : 2 bytes
//  AND-mask              : 2 bytes
//  OR-mask               : 2 bytes
// Response:
//  Function code         : 1 byte (0x16)
//  Reference address     : 2 bytes
//  AND-mask              : 2 bytes
//  OR-mask               : 2 bytes
func (sf *client) MaskWriteRegister(slaveID byte, address, andMask, orMask uint16) error {
	if slaveID > AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, AddressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeMaskWriteRegister,
		Data:     pduDataBlock(address, andMask, orMask),
	})

	switch {
	case err != nil:
		return err
	case len(response.Data) != 6:
		// Fixed response length
		return fmt.Errorf("modbus: response data size '%v' does not match expected '%v'",
			len(response.Data), 6)
	case binary.BigEndian.Uint16(response.Data) != address:
		return fmt.Errorf("modbus: response address '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data), address)
	case binary.BigEndian.Uint16(response.Data[2:]) != andMask:
		return fmt.Errorf("modbus: response AND-mask '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), andMask)
	case binary.BigEndian.Uint16(response.Data[4:]) != orMask:
		return fmt.Errorf("modbus: response OR-mask '%v' does not match request '%v'",
			binary.BigEndian.Uint16(response.Data[4:]), orMask)
	}
	return nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x17)
//  Read starting address : 2 bytes
//  Quantity to read      : 2 bytes
//  Write starting address: 2 bytes
//  Quantity to write     : 2 bytes
//  Write byte count      : 1 byte
//  Write registers value : N* bytes
// Response:
//  Function code         : 1 byte (0x17)
//  Byte count            : 1 byte
//  Read registers value  : Nx2 bytes
func (sf *client) ReadWriteMultipleRegistersBytes(slaveID byte, readAddress, readQuantity,
	writeAddress, writeQuantity uint16, value []byte) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	if readQuantity < ReadWriteOnReadRegQuantityMin || readQuantity > ReadWriteOnReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity to read '%v' must be between '%v' and '%v'",
			readQuantity, ReadWriteOnReadRegQuantityMin, ReadWriteOnReadRegQuantityMax)
	}
	if writeQuantity < ReadWriteOnWriteRegQuantityMin || writeQuantity > ReadWriteOnWriteRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity to write '%v' must be between '%v' and '%v'",
			writeQuantity, ReadWriteOnWriteRegQuantityMin, ReadWriteOnWriteRegQuantityMax)
	}

	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadWriteMultipleRegisters,
		Data:     pduDataBlockSuffix(value, readAddress, readQuantity, writeAddress, writeQuantity),
	})
	if err != nil {
		return nil, err
	}
	if int(response.Data[0]) != (len(response.Data) - 1) {
		return nil, fmt.Errorf("modbus: response data size '%v' does not match count '%v'",
			len(response.Data)-1, response.Data[0])
	}
	return response.Data[1:], nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x17)
//  Read starting address quantity: 2 bytes
//  Quantity to read      : 2 bytes
//  Write starting address: 2 bytes
//  Quantity to write     : 2 bytes
//  Write byte count      : 1 byte
//  Write registers value : N* bytes
// Response:
//  Function code         : 1 byte (0x17)
//  Byte count            : 1 byte
//  Read registers value  : N 2-bytes
func (sf *client) ReadWriteMultipleRegisters(slaveID byte, readAddress, readQuantity,
	writeAddress, writeQuantity uint16, value []byte) ([]uint16, error) {
	b, err := sf.ReadWriteMultipleRegistersBytes(slaveID, readAddress, readQuantity,
		writeAddress, writeQuantity, value)
	if err != nil {
		return nil, err
	}
	return bytes2Uint16(b), nil
}

// Request:
//  Slave Id              : 1 byte
//  Function code         : 1 byte (0x18)
//  FIFO pointer address  : 2 bytes
// Response:
//  Function code         : 1 byte (0x18)
//  Byte count            : 2 bytes  only include follow
//  FIFO count            : 2 bytes (<=31)
//  FIFO value register   : Nx2 bytes
func (sf *client) ReadFIFOQueue(slaveID byte, address uint16) ([]byte, error) {
	if slaveID < AddressMin || slaveID > AddressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressMin, AddressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadFIFOQueue,
		Data:     pduDataBlock(address),
	})
	switch {
	case err != nil:
		return nil, err
	case len(response.Data) < 4:
		return nil, fmt.Errorf("modbus: response data size '%v' is less than expected '%v'",
			len(response.Data), 4)
	case len(response.Data)-2 != int(binary.BigEndian.Uint16(response.Data)):
		return nil, fmt.Errorf("modbus: response data size '%v' does not match count '%v'",
			len(response.Data)-2, binary.BigEndian.Uint16(response.Data))
	case int(binary.BigEndian.Uint16(response.Data[2:])) > 31:
		return nil, fmt.Errorf("modbus: fifo count '%v' is greater than expected '%v'",
			binary.BigEndian.Uint16(response.Data[2:]), 31)
	}
	return response.Data[4:], nil
}

// pduDataBlock creates a sequence of uint16 data.
func pduDataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// pduDataBlockSuffix creates a sequence of uint16 data and append the suffix plus its length.
func pduDataBlockSuffix(suffix []byte, value ...uint16) []byte {
	length := 2 * len(value)
	data := make([]byte, length+1+len(suffix))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	data[length] = uint8(len(suffix))
	copy(data[length+1:], suffix)
	return data
}

// responseError response error
func responseError(response ProtocolDataUnit) error {
	mbError := &ExceptionError{}
	if response.Data != nil && len(response.Data) > 0 {
		mbError.ExceptionCode = response.Data[0]
	}
	return mbError
}

// bytes2Uint16 bytes conver to uint16 for register
func bytes2Uint16(buf []byte) []uint16 {
	result := make([]uint16, len(buf)/2)
	for i := 0; i < len(buf)/2; i++ {
		result[i] = binary.BigEndian.Uint16(buf[i*2:])
	}
	return result
}
