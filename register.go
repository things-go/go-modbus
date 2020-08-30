package modbus

// 本文件提供了寄存器的底层封装,并且是线程安全的,丰富的api满足基本需求

import (
	"bytes"
	"encoding/binary"
	"sync"
)

// NodeRegister 节点寄存器
type NodeRegister struct {
	rw                                  sync.RWMutex // 读写锁
	slaveID                             byte
	coilsAddrStart, coilsQuantity       uint16
	coils                               []uint8
	discreteAddrStart, discreteQuantity uint16
	discrete                            []uint8
	inputAddrStart                      uint16
	input                               []uint16
	holdingAddrStart                    uint16
	holding                             []uint16
}

// NewNodeRegister 创建一个modbus子节点寄存器列表
func NewNodeRegister(slaveID byte,
	coilsAddrStart, coilsQuantity,
	discreteAddrStart, discreteQuantity,
	inputAddrStart, inputQuantity,
	holdingAddrStart, holdingQuantity uint16) *NodeRegister {
	coilsBytes := (int(coilsQuantity) + 7) / 8
	discreteBytes := (int(discreteQuantity) + 7) / 8

	b := make([]byte, coilsBytes+discreteBytes)
	w := make([]uint16, int(inputQuantity)+int(holdingQuantity))
	return &NodeRegister{
		slaveID:           slaveID,
		coilsAddrStart:    coilsAddrStart,
		coilsQuantity:     coilsQuantity,
		coils:             b[:coilsBytes],
		discreteAddrStart: discreteAddrStart,
		discreteQuantity:  discreteQuantity,
		discrete:          b[coilsBytes:],
		inputAddrStart:    inputAddrStart,
		input:             w[:inputQuantity],
		holdingAddrStart:  holdingAddrStart,
		holding:           w[inputQuantity:],
	}
}

// SlaveID 获取从站地址
func (sf *NodeRegister) SlaveID() byte {
	sf.rw.RLock()
	id := sf.slaveID
	sf.rw.RUnlock()
	return id
}

// SetSlaveID 更改从站地址
func (sf *NodeRegister) SetSlaveID(id byte) *NodeRegister {
	sf.rw.Lock()
	sf.slaveID = id
	sf.rw.Unlock()
	return sf
}

// CoilsAddrParam 读coil起始地址与数量
func (sf *NodeRegister) CoilsAddrParam() (start, quantity uint16) {
	return sf.coilsAddrStart, sf.coilsQuantity
}

// DiscreteParam  读discrete起始地址与数量
func (sf *NodeRegister) DiscreteParam() (start, quantity uint16) {
	return sf.discreteAddrStart, sf.discreteQuantity
}

// InputAddrParam  读input起始地址与数量
func (sf *NodeRegister) InputAddrParam() (start, quantity uint16) {
	return sf.inputAddrStart, uint16(len(sf.input))
}

// HoldingAddrParam  读holding起始地址与数量
func (sf *NodeRegister) HoldingAddrParam() (start, quantity uint16) {
	return sf.holdingAddrStart, uint16(len(sf.holding))
}

// getBits 读取切片的位的值, nBits <= 8, nBits + start <= len(buf)*8
func getBits(buf []byte, start, nBits uint16) uint8 {
	byteOffset := start / 8         // 计算字节偏移量
	preBits := start - byteOffset*8 // 有多少个位需要设置

	mask := (uint16(1) << nBits) - 1 // 准备一个掩码来设置新的位
	word := uint16(buf[byteOffset])  // 复制到临时存储
	if preBits+nBits > 8 {
		word |= uint16(buf[byteOffset+1]) << 8
	}
	word >>= preBits // 抛弃不用的位
	word &= mask
	return uint8(word)
}

// setBits 设置切片的位的值, nBits <= 8, nBits + start <= len(buf)*8
func setBits(buf []byte, start, nBits uint16, value byte) {
	byteOffset := start / 8              // 计算字节偏移量
	preBits := start - byteOffset*8      // 有多少个位需要设置
	newValue := uint16(value) << preBits // 移到要设置的位的位置
	mask := uint16((1 << nBits) - 1)     // 准备一个掩码来设置新的位
	mask <<= preBits
	newValue &= mask
	word := uint16(buf[byteOffset]) // 复制到临时存储
	if (preBits + nBits) > 8 {
		word |= uint16(buf[byteOffset+1]) << 8
	}

	word = (word & (^mask)) | newValue   // 要写的位置清零
	buf[byteOffset] = uint8(word & 0xFF) // 写回到存储中
	if (preBits + nBits) > 8 {
		buf[byteOffset+1] = uint8(word >> 8)
	}
}

// WriteCoils 写线圈
func (sf *NodeRegister) WriteCoils(address, quality uint16, valBuf []byte) error {
	sf.rw.Lock()
	if len(valBuf)*8 >= int(quality) && (address >= sf.coilsAddrStart) &&
		((address + quality) <= (sf.coilsAddrStart + sf.coilsQuantity)) {
		start := address - sf.coilsAddrStart
		nCoils := int16(quality)
		for idx := 0; nCoils > 0; idx++ {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			setBits(sf.coils, start, uint16(num), valBuf[idx])
			start += 8
			nCoils -= 8
		}
		sf.rw.Unlock()
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteSingleCoil 写单个线圈
func (sf *NodeRegister) WriteSingleCoil(address uint16, val bool) error {
	newVal := byte(0)
	if val {
		newVal = 1
	}
	return sf.WriteCoils(address, 1, []byte{newVal})
}

// ReadCoils 读线圈,返回值
func (sf *NodeRegister) ReadCoils(address, quality uint16) ([]byte, error) {
	sf.rw.RLock()
	if (address >= sf.coilsAddrStart) &&
		((address + quality) <= (sf.coilsAddrStart + sf.coilsQuantity)) {
		start := address - sf.coilsAddrStart
		nCoils := int16(quality)
		result := make([]byte, 0, (quality+7)/8)
		for ; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			result = append(result, getBits(sf.coils, start, uint16(num)))
			start += 8
		}
		sf.rw.RUnlock()
		return result, nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadSingleCoil 读单个线圈
func (sf *NodeRegister) ReadSingleCoil(address uint16) (bool, error) {
	v, err := sf.ReadCoils(address, 1)
	if err != nil {
		return false, err
	}
	return v[0] > 0, nil
}

// WriteDiscretes 写离散量
func (sf *NodeRegister) WriteDiscretes(address, quality uint16, valBuf []byte) error {
	sf.rw.Lock()
	if len(valBuf)*8 >= int(quality) && (address >= sf.discreteAddrStart) &&
		((address + quality) <= (sf.discreteAddrStart + sf.discreteQuantity)) {
		start := address - sf.discreteAddrStart
		nCoils := int16(quality)
		for idx := 0; nCoils > 0; idx++ {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			setBits(sf.discrete, start, uint16(num), valBuf[idx])
			start += 8
			nCoils -= 8
		}
		sf.rw.Unlock()
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteSingleDiscrete 写单个离散量
func (sf *NodeRegister) WriteSingleDiscrete(address uint16, val bool) error {
	newVal := byte(0)
	if val {
		newVal = 1
	}
	return sf.WriteDiscretes(address, 1, []byte{newVal})
}

// ReadDiscretes 读离散量
func (sf *NodeRegister) ReadDiscretes(address, quality uint16) ([]byte, error) {
	sf.rw.RLock()
	if (address >= sf.discreteAddrStart) &&
		((address + quality) <= (sf.discreteAddrStart + sf.discreteQuantity)) {
		start := address - sf.discreteAddrStart
		nCoils := int16(quality)
		result := make([]byte, 0, (quality+7)/8)
		for ; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			result = append(result, getBits(sf.discrete, start, uint16(num)))
			start += 8
		}
		sf.rw.RUnlock()
		return result, nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadSingleDiscrete 读单个离散量
func (sf *NodeRegister) ReadSingleDiscrete(address uint16) (bool, error) {
	v, err := sf.ReadDiscretes(address, 1)
	if err != nil {
		return false, err
	}
	return v[0] > 0, nil
}

// WriteHoldingsBytes 写保持寄存器
func (sf *NodeRegister) WriteHoldingsBytes(address, quality uint16, valBuf []byte) error {
	sf.rw.Lock()
	if len(valBuf) == int(quality*2) &&
		(address >= sf.holdingAddrStart) &&
		((address + quality) <= (sf.holdingAddrStart + uint16(len(sf.holding)))) {
		start := address - sf.holdingAddrStart
		end := start + quality
		buf := bytes.NewBuffer(valBuf)
		err := binary.Read(buf, binary.BigEndian, sf.holding[start:end])
		sf.rw.Unlock()
		if err != nil {
			return &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteHoldings 写保持寄存器
func (sf *NodeRegister) WriteHoldings(address uint16, valBuf []uint16) error {
	quality := uint16(len(valBuf))
	sf.rw.Lock()
	if (address >= sf.holdingAddrStart) &&
		((address + quality) <= (sf.holdingAddrStart + uint16(len(sf.holding)))) {
		start := address - sf.holdingAddrStart
		end := start + quality
		copy(sf.holding[start:end], valBuf)
		sf.rw.Unlock()
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadHoldingsBytes 读保持寄存器,仅返回寄存器值
func (sf *NodeRegister) ReadHoldingsBytes(address, quality uint16) ([]byte, error) {
	sf.rw.RLock()
	if (address >= sf.holdingAddrStart) &&
		((address + quality) <= (sf.holdingAddrStart + uint16(len(sf.holding)))) {
		start := address - sf.holdingAddrStart
		end := start + quality
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, sf.holding[start:end])
		sf.rw.RUnlock()
		if err != nil {
			return nil, &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return buf.Bytes(), nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadHoldings 读保持寄存器,仅返回寄存器值
func (sf *NodeRegister) ReadHoldings(address, quality uint16) ([]uint16, error) {
	sf.rw.RLock()
	if (address >= sf.holdingAddrStart) &&
		((address + quality) <= (sf.holdingAddrStart + uint16(len(sf.holding)))) {
		start := address - sf.holdingAddrStart
		end := start + quality
		result := make([]uint16, quality)
		copy(result, sf.holding[start:end])
		sf.rw.RUnlock()
		return result, nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteInputsBytes 写输入寄存器
func (sf *NodeRegister) WriteInputsBytes(address, quality uint16, regBuf []byte) error {
	sf.rw.Lock()
	if len(regBuf) == int(quality*2) &&
		(address >= sf.inputAddrStart) &&
		((address + quality) <= (sf.inputAddrStart + uint16(len(sf.input)))) {
		start := address - sf.inputAddrStart
		end := start + quality
		buf := bytes.NewBuffer(regBuf)
		err := binary.Read(buf, binary.BigEndian, sf.input[start:end])
		sf.rw.Unlock()
		if err != nil {
			return &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteInputs 写输入寄存器
func (sf *NodeRegister) WriteInputs(address uint16, valBuf []uint16) error {
	quality := uint16(len(valBuf))
	sf.rw.Lock()
	if (address >= sf.inputAddrStart) &&
		((address + quality) <= (sf.inputAddrStart + uint16(len(sf.input)))) {
		start := address - sf.inputAddrStart
		end := start + quality
		copy(sf.input[start:end], valBuf)
		sf.rw.Unlock()
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadInputsBytes 读输入寄存器
func (sf *NodeRegister) ReadInputsBytes(address, quality uint16) ([]byte, error) {
	sf.rw.RLock()
	if (address >= sf.inputAddrStart) &&
		((address + quality) <= (sf.inputAddrStart + uint16(len(sf.input)))) {
		start := address - sf.inputAddrStart
		end := start + quality
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, sf.input[start:end])
		sf.rw.RUnlock()
		if err != nil {
			return nil, &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return buf.Bytes(), nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadInputs 读输入寄存器
func (sf *NodeRegister) ReadInputs(address, quality uint16) ([]uint16, error) {
	sf.rw.RLock()
	if (address >= sf.inputAddrStart) &&
		((address + quality) <= (sf.inputAddrStart + uint16(len(sf.input)))) {
		start := address - sf.inputAddrStart
		end := start + quality
		result := make([]uint16, quality)
		copy(result, sf.input[start:end])
		sf.rw.RUnlock()
		return result, nil
	}
	sf.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// MaskWriteHolding 屏蔽写保持寄存器 (val & andMask) | (orMask & ^andMask)
func (sf *NodeRegister) MaskWriteHolding(address, andMask, orMask uint16) error {
	sf.rw.Lock()
	if (address >= sf.holdingAddrStart) &&
		((address + 1) <= (sf.holdingAddrStart + uint16(len(sf.holding)))) {
		sf.holding[address] &= andMask
		sf.holding[address] |= orMask & ^andMask
		sf.rw.Unlock()
		return nil
	}
	sf.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}
