package modbus

import (
	"encoding/binary"
)

const (
	funcReadMinSize       = 4 // 读操作 最小数据域个数
	funcWriteMinSize      = 4 // 写操作 最小数据域个数
	funcWriteMultiMinSize = 5 // 写多个操作 最小数据域个数
	funcReadWriteMinSize  = 9 // 读写操作 最小数据域个数
	funcMaskWriteMinSize  = 6 // 屏蔽写操作 最小数据域个数
)

// FunctionHandler 功能码对应的函数回调
type FunctionHandler func(reg *NodeRegister, data []byte) ([]byte, error)

type serverHandler struct {
	function map[uint8]FunctionHandler
}

func newServerHandler() *serverHandler {
	return &serverHandler{
		function: map[uint8]FunctionHandler{
			FuncCodeReadDiscreteInputs:         funcReadDiscreteInputs,
			FuncCodeReadCoils:                  funcReadCoils,
			FuncCodeWriteSingleCoil:            funcWriteSingleCoil,
			FuncCodeWriteMultipleCoils:         funcWriteMultiCoils,
			FuncCodeReadInputRegisters:         funcReadInputRegisters,
			FuncCodeReadHoldingRegisters:       funcReadHoldingRegisters,
			FuncCodeWriteSingleRegister:        funcWriteSingleRegister,
			FuncCodeWriteMultipleRegisters:     funcWriteMultiHoldingRegisters,
			FuncCodeReadWriteMultipleRegisters: funcReadWriteMultiHoldingRegisters,
			FuncCodeMaskWriteRegister:          funcMaskWriteRegisters,
			// funcCodeReadFIFOQueue:
		},
	}
}

// RegisterFunctionHandler 注册回调函数
func (this *serverHandler) RegisterFunctionHandler(funcCode uint8, function FunctionHandler) {
	this.function[funcCode] = function
}

// readBits 读位寄存器
func readBits(reg *NodeRegister, data []byte, isCoil bool) ([]byte, error) {
	var value []byte
	var err error

	if len(data) != funcReadMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	quality := binary.BigEndian.Uint16(data[2:])
	if quality < ReadBitsQuantityMin || quality > ReadBitsQuantityMax {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}
	if isCoil {
		value, err = reg.ReadCoils(address, quality)
	} else {
		value, err = reg.ReadDiscretes(address, quality)
	}
	if err != nil {
		return nil, err
	}
	result := make([]byte, 0, len(value)+1)
	result = append(result, byte(len(value)))
	return append(result, value...), nil
}

// funcReadDiscreteInputs 读离散量输入,返回仅含PDU数据域
func funcReadDiscreteInputs(reg *NodeRegister, data []byte) ([]byte, error) {
	return readBits(reg, data, false)
}

// funcReadCoils 读线圈,返回仅含PDU数据域
func funcReadCoils(reg *NodeRegister, data []byte) ([]byte, error) {
	return readBits(reg, data, true)
}

// funcWriteSingleCoil 写单个线圈,返回仅含PDU数据域
func funcWriteSingleCoil(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != funcWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	newValue := binary.BigEndian.Uint16(data[2:])
	if !(newValue == 0xFF00 || newValue == 0x0000) {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}

	}
	b := byte(0)
	if newValue == 0xFF00 {
		b = 1
	}
	err := reg.WriteCoils(address, 1, []byte{b})
	return data, err
}

// funcWriteMultiCoils 写多个线圈,返回仅含PDU数据域
func funcWriteMultiCoils(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < funcWriteMultiMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	quality := binary.BigEndian.Uint16(data[2:])
	byteCnt := data[4]
	if quality < WriteBitsQuantityMin || quality > WriteBitsQuantityMax ||
		byteCnt != byte((quality+7)/8) {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}
	err := reg.WriteCoils(address, quality, data[5:])
	return data[:4], err
}

// readRegisters 读继寄器,返回仅含PDU数据域
func readRegisters(reg *NodeRegister, data []byte, isHolding bool) ([]byte, error) {
	var err error
	var value []byte

	if len(data) != funcReadMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	quality := binary.BigEndian.Uint16(data[2:])
	if quality > ReadRegQuantityMax || quality < ReadRegQuantityMin {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	if isHolding {
		value, err = reg.ReadHoldingsBytes(address, quality)
	} else {
		value, err = reg.ReadInputsBytes(address, quality)
	}
	if err != nil {
		return nil, err
	}
	result := make([]byte, 0, len(value)+1)
	result = append(result, byte(quality*2))
	result = append(result, value...)
	return result, nil
}

// funcReadInputRegisters 读输入寄存器,返回仅含PDU数据域
func funcReadInputRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	return readRegisters(reg, data, false)
}

// funcReadHoldingRegisters 读保持寄存器,返回仅含PDU数据域
func funcReadHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	return readRegisters(reg, data, true)
}

// funcWriteSingleRegister 写单个保持寄存器,返回仅含PDU数据域
func funcWriteSingleRegister(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != funcWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	err := reg.WriteHoldingsBytes(address, 1, data[2:])
	return data, err
}

// funcWriteMultiHoldingRegisters 写多个保持寄存器,返回仅含PDU数据域
func funcWriteMultiHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < funcWriteMultiMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	count := binary.BigEndian.Uint16(data[2:])
	byteCnt := data[4]
	if count < WriteRegQuantityMin || count > WriteRegQuantityMax ||
		byteCnt != uint8(count*2) {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	err := reg.WriteHoldingsBytes(address, count, data[5:])
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(data[2:], count)
	return data[:4], nil
}

// funcReadWriteMultiHoldingRegisters 读写多个保持寄存器,返回仅含PDU数据域
func funcReadWriteMultiHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < funcReadWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	readAddress := binary.BigEndian.Uint16(data)
	readCount := binary.BigEndian.Uint16(data[2:])
	writeAddress := binary.BigEndian.Uint16(data[4:])
	WriteCount := binary.BigEndian.Uint16(data[6:])
	writeByteCnt := data[8]
	if readCount < ReadWriteOnReadRegQuantityMin || readCount > ReadWriteOnReadRegQuantityMax ||
		WriteCount < ReadWriteOnWriteRegQuantityMin || WriteCount > ReadWriteOnWriteRegQuantityMax ||
		writeByteCnt != uint8(WriteCount*2) {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	if err := reg.WriteHoldingsBytes(writeAddress, WriteCount, data[9:]); err != nil {
		return nil, err
	}
	value, err := reg.ReadHoldingsBytes(readAddress, readCount)
	if err != nil {
		return nil, err
	}
	result := make([]byte, 0, len(value)+1)
	result = append(result, byte(readCount*2))
	result = append(result, value...)
	return result, nil
}

// funcMaskWriteRegisters 屏蔽写寄存器,返回仅含PDU数据域
func funcMaskWriteRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != funcMaskWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	referAddress := binary.BigEndian.Uint16(data)
	andMask := binary.BigEndian.Uint16(data[2:])
	orMask := binary.BigEndian.Uint16(data[4:])
	err := reg.MaskWriteHolding(referAddress, andMask, orMask)
	return data, err
}

// TODO funcReadFIFOQueue,返回仅含PDU数据域
// func (this *ExtraOption)funcReadFIFOQueue(*NodeRegister, []byte) ([]byte, error) {
// 	return nil, nil
// }
