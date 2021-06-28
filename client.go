package modbus

import (
	"encoding/binary"
	"fmt"
)

// check implements Client interface.
var _ Client = (*client)(nil)

// Option custom option
type Option func(c *client)

// WithAddressMin set custom address max value, default AddressMin
func WithAddressMin(v byte) Option {
	return func(c *client) {
		c.addressMin = v
	}
}

// WithAddressMax set custom address max value, default AddressMax
func WithAddressMax(v byte) Option {
	return func(c *client) {
		c.addressMax = v
	}
}

// client implements Client interface.
type client struct {
	ClientProvider
	addressMin byte
	addressMax byte
}

// NewClient creates a new modbus client with given backend handler.
// default proto address limit is 1~247 AddressMax
// you can change with custom option.
// // when your device have address upon addressMax
func NewClient(p ClientProvider, opts ...Option) Client {
	c := &client{p, AddressMin, AddressMax}
	for _, opt := range opts {
		opt(c)
	}
	return c
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
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	if quantity < ReadBitsQuantityMin || quantity > ReadBitsQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadBitsQuantityMin, ReadBitsQuantityMax)
	}

	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCodeReadCoils,
		uint162Bytes(address, quantity),
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
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	if quantity < ReadBitsQuantityMin || quantity > ReadBitsQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadBitsQuantityMin, ReadBitsQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadDiscreteInputs,
		Data:     uint162Bytes(address, quantity),
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
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
// Response:
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
func (sf *client) WriteSingleCoil(slaveID byte, address uint16, isOn bool) error {
	if slaveID > sf.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, sf.addressMax)
	}
	var value uint16
	if isOn { // The requested ON/OFF state can only be 0xFF00 and 0x0000
		value = 0xFF00
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteSingleCoil,
		Data:     uint162Bytes(address, value),
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
	if slaveID > sf.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, sf.addressMax)
	}
	if quantity < WriteBitsQuantityMin || quantity > WriteBitsQuantityMax {
		return fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, WriteBitsQuantityMin, WriteBitsQuantityMax)
	}

	if len(value)*8 < int(quantity) {
		return fmt.Errorf("modbus: value bits size '%v' does not greater or equal to quantity '%v'", len(value)*8, quantity)
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

/*********************************16-bits**************************************/

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
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	if quantity < ReadRegQuantityMin || quantity > ReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadRegQuantityMin, ReadRegQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadInputRegisters,
		Data:     uint162Bytes(address, quantity),
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
//  Function code         : 1 byte (0x03)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x03)
//  Byte count            : 1 byte
//  Register value        : Nx2 bytes
func (sf *client) ReadHoldingRegistersBytes(slaveID byte, address, quantity uint16) ([]byte, error) {
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	if quantity < ReadRegQuantityMin || quantity > ReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, ReadRegQuantityMin, ReadRegQuantityMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadHoldingRegisters,
		Data:     uint162Bytes(address, quantity),
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
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
// Response:
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
func (sf *client) WriteSingleRegister(slaveID byte, address, value uint16) error {
	if slaveID > sf.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, sf.addressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeWriteSingleRegister,
		Data:     uint162Bytes(address, value),
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
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Registers value       : N* bytes
// Response:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (sf *client) WriteMultipleRegistersBytes(slaveID byte, address, quantity uint16, value []byte) error {
	if slaveID > sf.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, sf.addressMax)
	}
	if quantity < WriteRegQuantityMin || quantity > WriteRegQuantityMax {
		return fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v'",
			quantity, WriteRegQuantityMin, WriteRegQuantityMax)
	}

	if len(value) != int(quantity*2) {
		return fmt.Errorf("modbus: value length '%v' does not twice as quantity '%v'", len(value), quantity)
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
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Registers value       : N* bytes
// Response:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (sf *client) WriteMultipleRegisters(slaveID byte, address, quantity uint16, value []uint16) error {
	return sf.WriteMultipleRegistersBytes(slaveID, address, quantity, uint162Bytes(value...))
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
	if slaveID > sf.addressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, AddressBroadCast, sf.addressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeMaskWriteRegister,
		Data:     uint162Bytes(address, andMask, orMask),
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
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	if readQuantity < ReadWriteOnReadRegQuantityMin || readQuantity > ReadWriteOnReadRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity to read '%v' must be between '%v' and '%v'",
			readQuantity, ReadWriteOnReadRegQuantityMin, ReadWriteOnReadRegQuantityMax)
	}
	if writeQuantity < ReadWriteOnWriteRegQuantityMin || writeQuantity > ReadWriteOnWriteRegQuantityMax {
		return nil, fmt.Errorf("modbus: quantity to write '%v' must be between '%v' and '%v'",
			writeQuantity, ReadWriteOnWriteRegQuantityMin, ReadWriteOnWriteRegQuantityMax)
	}

	if len(value) != int(writeQuantity*2) {
		return nil, fmt.Errorf("modbus: value length '%v' does not twice as write quantity '%v'",
			len(value), writeQuantity)
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
	if slaveID < sf.addressMin || slaveID > sf.addressMax {
		return nil, fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			slaveID, sf.addressMin, sf.addressMax)
	}
	response, err := sf.Send(slaveID, ProtocolDataUnit{
		FuncCode: FuncCodeReadFIFOQueue,
		Data:     uint162Bytes(address),
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

// uint162Bytes creates a sequence of uint16 data.
func uint162Bytes(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// bytes2Uint16 bytes convert to uint16 for register.
func bytes2Uint16(buf []byte) []uint16 {
	data := make([]uint16, 0, len(buf)/2)
	for i := 0; i < len(buf)/2; i++ {
		data = append(data, binary.BigEndian.Uint16(buf[i*2:]))
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

// responseError response error.
func responseError(response ProtocolDataUnit) error {
	mbError := &ExceptionError{}
	if response.Data != nil && len(response.Data) > 0 {
		mbError.ExceptionCode = response.Data[0]
	}
	return mbError
}
