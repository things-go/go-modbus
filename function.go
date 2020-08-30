package modbus

import (
	"encoding/binary"
	"errors"
	"sync"
)

// handle pdu data filed limit size.
const (
	FuncReadMinSize       = 4 // 读操作 最小数据域个数
	FuncWriteMinSize      = 4 // 写操作 最小数据域个数
	FuncWriteMultiMinSize = 5 // 写多个操作 最小数据域个数
	FuncReadWriteMinSize  = 9 // 读写操作 最小数据域个数
	FuncMaskWriteMinSize  = 6 // 屏蔽写操作 最小数据域个数
)

// FunctionHandler 功能码对应的函数回调.
// data 仅pdu数据域 不含功能码, return pdu 数据域,不含功能码.
type FunctionHandler func(reg *NodeRegister, data []byte) ([]byte, error)

type serverCommon struct {
	node     sync.Map
	function map[uint8]FunctionHandler
}

func newServerCommon() *serverCommon {
	return &serverCommon{
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
		},
	}
}

// AddNodes 增加节点.
func (sf *serverCommon) AddNodes(nodes ...*NodeRegister) {
	for _, v := range nodes {
		sf.node.Store(v.slaveID, v)
	}
}

// DeleteNode 删除一个节点.
func (sf *serverCommon) DeleteNode(slaveID byte) {
	sf.node.Delete(slaveID)
}

// DeleteAllNode 删除所有节点.
func (sf *serverCommon) DeleteAllNode() {
	sf.node.Range(func(k, v interface{}) bool {
		sf.node.Delete(k)
		return true
	})
}

// GetNode 获取一个节点.
func (sf *serverCommon) GetNode(slaveID byte) (*NodeRegister, error) {
	v, ok := sf.node.Load(slaveID)
	if !ok {
		return nil, errors.New("slaveID not exist")
	}
	return v.(*NodeRegister), nil
}

// GetNodeList 获取节点列表.
func (sf *serverCommon) GetNodeList() []*NodeRegister {
	list := make([]*NodeRegister, 0)
	sf.node.Range(func(k, v interface{}) bool {
		list = append(list, v.(*NodeRegister))
		return true
	})
	return list
}

// Range 扫描节点 same as sync map range.
func (sf *serverCommon) Range(f func(slaveID byte, node *NodeRegister) bool) {
	sf.node.Range(func(k, v interface{}) bool {
		return f(k.(byte), v.(*NodeRegister))
	})
}

// RegisterFunctionHandler 注册回调函数.
func (sf *serverCommon) RegisterFunctionHandler(funcCode uint8, function FunctionHandler) {
	if function != nil {
		sf.function[funcCode] = function
	}
}

// readBits 读位寄存器.
func readBits(reg *NodeRegister, data []byte, isCoil bool) ([]byte, error) {
	var value []byte
	var err error

	if len(data) != FuncReadMinSize {
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

// funcReadDiscreteInputs 读离散量输入,返回仅含PDU数据域.
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  return:
//  Byte count            : 1 bytes
//  Coils status          : n bytes  n = Quantity/8 or n = Quantity/8 + 1
func funcReadDiscreteInputs(reg *NodeRegister, data []byte) ([]byte, error) {
	return readBits(reg, data, false)
}

// funcReadCoils read multi coils.
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  return:
//  Byte count            : 1 bytes
//  Coils status          : n bytes  n = Quantity/8 or n = Quantity/8 + 1
func funcReadCoils(reg *NodeRegister, data []byte) ([]byte, error) {
	return readBits(reg, data, true)
}

// funcWriteSingleCoil write single coil.
// data:
//  Address      		  : 2 byte
//  Value                 : 2 byte  0xff00 or 0x0000s
//  return:
//  Address      		  : 2 byte
//  Value                 : 2 byte  0xff00 or 0x0000
func funcWriteSingleCoil(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != FuncWriteMinSize {
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

// funcWriteMultiCoils write multi coils.
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  Byte count            : 1 byte
//  Value                 : n byte
//  return:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
func funcWriteMultiCoils(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < FuncWriteMultiMinSize {
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

// readRegisters read multi registers.
func readRegisters(reg *NodeRegister, data []byte, isHolding bool) ([]byte, error) {
	var err error
	var value []byte

	if len(data) != FuncReadMinSize {
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

// funcReadInputRegisters 读输入寄存器
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  return:
//  Byte count            : 2 byte  Quantity*2
//  Value                 : (Quantity)*2 byte
func funcReadInputRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	return readRegisters(reg, data, false)
}

// funcReadHoldingRegisters 读保持寄存器
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  return:
//  Byte count            : 2 byte  Quantity*2
//  Value                 : (Quantity)*2 byte
func funcReadHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	return readRegisters(reg, data, true)
}

// funcWriteSingleRegister 写单个保持寄存器
// data:
//  Address      		: 2 byte
//  Value              	: 2 byte
//  return:
//  Address            	: 2 byte
//  Value               : 2 byte
func funcWriteSingleRegister(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != FuncWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	err := reg.WriteHoldingsBytes(address, 1, data[2:])
	return data, err
}

// funcWriteMultiHoldingRegisters 写多个保持寄存器
// data:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
//  Byte count            : 1 byte Quantity*2
//  Value                 : Quantity*2 byte
//  return:
//  Starting address      : 2 byte
//  Quantity              : 2 byte
func funcWriteMultiHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < FuncWriteMultiMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	address := binary.BigEndian.Uint16(data)
	count := binary.BigEndian.Uint16(data[2:])
	byteCnt := data[4]
	if count < WriteRegQuantityMin || count > WriteRegQuantityMax ||
		uint16(byteCnt) != count*2 {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	err := reg.WriteHoldingsBytes(address, count, data[5:])
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(data[2:], count)
	return data[:4], nil
}

// funcReadWriteMultiHoldingRegisters 读写多个保持寄存器
// data:
//  Read Starting address       : 2 byte
//  Quantity read               : 2 byte
//  Write Starting address      : 2 byte
//  Quantity Write              : 2 byte
//  Byte count Write            : 1 byte (Quantity Write)*2
//  Value Write                 : (Quantity Write)*2 byte
//  return:
//  Byte count            : 2 byte  (Quantity read)*2
//  Value                 : (Quantity read)*2 byte
func funcReadWriteMultiHoldingRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) < FuncReadWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	readAddress := binary.BigEndian.Uint16(data)
	readCount := binary.BigEndian.Uint16(data[2:])
	writeAddress := binary.BigEndian.Uint16(data[4:])
	WriteCount := binary.BigEndian.Uint16(data[6:])
	writeByteCnt := data[8]
	if readCount < ReadWriteOnReadRegQuantityMin || readCount > ReadWriteOnReadRegQuantityMax ||
		WriteCount < ReadWriteOnWriteRegQuantityMin || WriteCount > ReadWriteOnWriteRegQuantityMax ||
		uint16(writeByteCnt) != WriteCount*2 {
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

// funcMaskWriteRegisters 屏蔽写寄存器
// data:
//  address				  : 2 byte
//  And_mask              : 2 byte
//  Or_mask               : 2 byte
//  return:
//  address				  : 2 byte
//  And_mask              : 2 byte
//  Or_mask               : 2 byte
func funcMaskWriteRegisters(reg *NodeRegister, data []byte) ([]byte, error) {
	if len(data) != FuncMaskWriteMinSize {
		return nil, &ExceptionError{ExceptionCodeIllegalDataValue}
	}

	referAddress := binary.BigEndian.Uint16(data)
	andMask := binary.BigEndian.Uint16(data[2:])
	orMask := binary.BigEndian.Uint16(data[4:])
	err := reg.MaskWriteHolding(referAddress, andMask, orMask)
	return data, err
}

// TODO funcReadFIFOQueue
// func (this *ExtraOption)funcReadFIFOQueue(*NodeRegister, []byte) ([]byte, error) {
// 	return nil, nil
// }
