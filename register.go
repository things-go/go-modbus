package modbus

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

// NewNodeRegister 创建一个modbus子节点
func NewNodeRegister(slaveID byte,
	coilsAddrStart, coilsQuantity uint16, coils []uint8,
	discreteAddrStart, discreteQuantity uint16, discrete []uint8,
	inputAddrStart uint16, input []uint16,
	holdingAddrStart uint16, holding []uint16) *NodeRegister {
	return &NodeRegister{
		slaveID:           slaveID,
		coilsAddrStart:    coilsAddrStart,
		coilsQuantity:     coilsQuantity,
		coils:             coils,
		discreteAddrStart: discreteAddrStart,
		discreteQuantity:  discreteQuantity,
		discrete:          discrete,
		inputAddrStart:    inputAddrStart,
		input:             input,
		holdingAddrStart:  holdingAddrStart,
		holding:           holding,
	}
}

// getBits 读取切片的位的值, nBits <= 8, nBits + start <= len(buf)*8
func getBits(buf []byte, start, nBits uint16) uint8 {
	byteOffset := start / 8         // 计算字节偏移量
	preBits := start - byteOffset*8 // 有多少个位需要设置

	mask := uint16(uint16(1)<<nBits) - 1 // 准备一个掩码来设置新的位
	word := uint16(buf[byteOffset])      // 复制到临时存储
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
	word := uint16(buf[byteOffset]) // 复制到临时存储
	if (preBits + nBits) > 8 {
		word |= uint16(buf[byteOffset+1]) << 8
	}

	word = uint16((word & (^mask)) | uint16(newValue)) // 要写的位置清零
	buf[byteOffset] = uint8(word & 0xFF)               // 写回到存储中
	if (preBits + nBits) > 8 {
		buf[byteOffset+1] = uint8(word >> 8)
	}
}

// SlaveID 获取从站地址
func (this *NodeRegister) SlaveID() byte {
	this.rw.RLock()
	id := this.slaveID
	this.rw.RUnlock()
	return id
}

// WriteCoils 写线圈
func (this *NodeRegister) WriteCoils(address, quality uint16, valBuf []byte) error {
	this.rw.Lock()
	if len(valBuf)*8 >= int(quality) && (address >= this.coilsAddrStart) &&
		((address + quality) <= (this.coilsAddrStart + this.coilsQuantity)) {
		start := address - this.coilsAddrStart
		nCoils := int16(quality)
		for idx := 0; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			setBits(this.coils, start, uint16(num), valBuf[idx])
			start += 8
		}
		this.rw.Unlock()
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteSingleCoil 写单个线圈
func (this *NodeRegister) WriteSingleCoil(address uint16, val bool) error {
	newVal := byte(0)
	if val {
		newVal = 1
	}
	return this.WriteCoils(address, 1, []byte{newVal})
}

// ReadCoils 读线圈,返回值
func (this *NodeRegister) ReadCoils(address, quality uint16) ([]byte, error) {
	this.rw.RLock()
	if (address >= this.coilsAddrStart) &&
		((address + quality) <= (this.coilsAddrStart + this.coilsQuantity)) {
		start := address - this.coilsAddrStart
		nCoils := int16(quality)
		result := make([]byte, 0, (quality+7)/8)
		for ; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			result = append(result, getBits(this.coils, start, uint16(num)))
			start += 8
		}
		this.rw.RUnlock()
		return result, nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadSingleCoil 读单个线圈
func (this *NodeRegister) ReadSingleCoil(address uint16) (bool, error) {
	v, err := this.ReadCoils(address, 1)
	if err != nil {
		return false, err
	}
	return v[0] > 0, nil
}

// WriteDiscretes 写离散量
func (this *NodeRegister) WriteDiscretes(address, quality uint16, valBuf []byte) error {
	this.rw.Lock()
	if len(valBuf)*8 >= int(quality) && (address >= this.discreteAddrStart) &&
		((address + quality) <= (this.discreteAddrStart + this.discreteQuantity)) {
		start := address - this.discreteAddrStart
		nCoils := int16(quality)
		for idx := 0; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			setBits(this.discrete, start, uint16(num), valBuf[idx])
			start += 8
		}
		this.rw.Unlock()
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteSingleDiscrete 写单个离散量
func (this *NodeRegister) WriteSingleDiscrete(address uint16, val bool) error {
	newVal := byte(0)
	if val {
		newVal = 1
	}
	return this.WriteDiscretes(address, 1, []byte{newVal})
}

// ReadDiscretes 读离散量
func (this *NodeRegister) ReadDiscretes(address, quality uint16) ([]byte, error) {
	this.rw.RLock()
	if (address >= this.discreteAddrStart) &&
		((address + quality) <= (this.discreteAddrStart + this.discreteQuantity)) {
		start := address - this.discreteAddrStart
		nCoils := int16(quality)
		result := make([]byte, 0, (quality+7)/8)
		for ; nCoils > 0; nCoils -= 8 {
			num := nCoils
			if nCoils > 8 {
				num = 8
			}
			result = append(result, getBits(this.discrete, start, uint16(num)))
			start += 8
		}
		this.rw.RUnlock()
		return result, nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadSingleDiscrete 读单个离散量
func (this *NodeRegister) ReadSingleDiscrete(address uint16) (bool, error) {
	v, err := this.ReadDiscretes(address, 1)
	if err != nil {
		return false, err
	}
	return v[0] > 0, nil
}

// WriteHoldingsBytes 写保持寄存器
func (this *NodeRegister) WriteHoldingsBytes(address, quality uint16, valBuf []byte) error {
	this.rw.Lock()
	if len(valBuf) == int(quality*2) &&
		(address >= this.holdingAddrStart) &&
		((address + quality) <= (this.holdingAddrStart + uint16(len(this.holding)))) {
		start := address - this.holdingAddrStart
		end := start + quality
		buf := bytes.NewBuffer(valBuf)
		err := binary.Read(buf, binary.BigEndian, this.holding[start:end])
		this.rw.Unlock()
		if err != nil {
			return &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteHoldings 写保持寄存器
func (this *NodeRegister) WriteHoldings(address uint16, valBuf []uint16) error {
	quality := uint16(len(valBuf))
	this.rw.Lock()
	if (address >= this.holdingAddrStart) &&
		((address + quality) <= (this.holdingAddrStart + uint16(len(this.holding)))) {
		start := address - this.holdingAddrStart
		end := start + quality
		copy(this.holding[start:end], valBuf)
		this.rw.Unlock()
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadHoldingsBytes 读保持寄存器,仅返回寄存器值
func (this *NodeRegister) ReadHoldingsBytes(address, quality uint16) ([]byte, error) {
	this.rw.RLock()
	if (address >= this.holdingAddrStart) &&
		((address + quality) <= (this.holdingAddrStart + uint16(len(this.holding)))) {
		start := address - this.holdingAddrStart
		end := start + quality
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, this.holding[start:end])
		this.rw.RUnlock()
		if err != nil {
			return nil, &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return buf.Bytes(), nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadHoldings 读保持寄存器,仅返回寄存器值
func (this *NodeRegister) ReadHoldings(address, quality uint16) ([]uint16, error) {
	this.rw.RLock()
	if (address >= this.holdingAddrStart) &&
		((address + quality) <= (this.holdingAddrStart + uint16(len(this.holding)))) {
		start := address - this.holdingAddrStart
		end := start + quality
		result := make([]uint16, 0, quality)
		copy(result, this.holding[start:end])
		this.rw.RUnlock()
		return result, nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteInputsBytes 写输入寄存器
func (this *NodeRegister) WriteInputsBytes(address, quality uint16, regBuf []byte) error {
	this.rw.Lock()
	if len(regBuf) == int(quality*2) &&
		(address >= this.inputAddrStart) &&
		((address + quality) <= (this.inputAddrStart + uint16(len(this.input)))) {
		start := address - this.inputAddrStart
		end := start + quality
		buf := bytes.NewBuffer(regBuf)
		err := binary.Read(buf, binary.BigEndian, this.input[start:end])
		this.rw.Unlock()
		if err != nil {
			return &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// WriteInputs 写输入寄存器
func (this *NodeRegister) WriteInputs(address uint16, valBuf []uint16) error {
	quality := uint16(len(valBuf))
	this.rw.Lock()
	if (address >= this.inputAddrStart) &&
		((address + quality) <= (this.inputAddrStart + uint16(len(this.input)))) {
		start := address - this.inputAddrStart
		end := start + quality
		copy(this.input[start:end], valBuf)
		this.rw.Unlock()
		return nil
	}
	this.rw.Unlock()
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadInputsBytes 读输入寄存器
func (this *NodeRegister) ReadInputsBytes(address, quality uint16) ([]byte, error) {
	this.rw.RLock()
	if (address >= this.inputAddrStart) &&
		((address + quality) <= (this.inputAddrStart + uint16(len(this.input)))) {
		start := address - this.inputAddrStart
		end := start + quality
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, this.input[start:end])
		this.rw.RUnlock()
		if err != nil {
			return nil, &ExceptionError{ExceptionCodeServerDeviceFailure}
		}
		return buf.Bytes(), nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// ReadInputs 读输入寄存器
func (this *NodeRegister) ReadInputs(address, quality uint16) ([]uint16, error) {
	this.rw.RLock()
	if (address >= this.inputAddrStart) &&
		((address + quality) <= (this.inputAddrStart + uint16(len(this.input)))) {
		start := address - this.inputAddrStart
		end := start + quality
		result := make([]uint16, 0, quality)
		copy(result, this.input[start:end])
		this.rw.RUnlock()
		return result, nil
	}
	this.rw.RUnlock()
	return nil, &ExceptionError{ExceptionCodeIllegalDataAddress}
}

// MaskWriteHolding 屏蔽写保持寄存器 (val & andMask) | (orMask & ^andMask)
func (this *NodeRegister) MaskWriteHolding(address, andMask, orMask uint16) error {
	this.rw.Lock()
	defer this.rw.Unlock()
	if (address >= this.holdingAddrStart) &&
		((address + 1) <= (this.holdingAddrStart + uint16(len(this.holding)))) {
		this.holding[address] &= andMask
		this.holding[address] |= orMask & ^andMask
		return nil
	}
	return &ExceptionError{ExceptionCodeIllegalDataAddress}
}
