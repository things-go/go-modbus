package modbus

// Client interface.
type Client interface {
	ClientProvider
	// Bits

	// ReadCoils reads from 1 to 2000 contiguous status of coils in a
	// remote device and returns coil status.
	ReadCoils(slaveID byte, address, quantity uint16) (results []byte, err error)
	// ReadDiscreteInputs reads from 1 to 2000 contiguous status of
	// discrete inputs in a remote device and returns input status.
	ReadDiscreteInputs(slaveID byte, address, quantity uint16) (results []byte, err error)

	// WriteSingleCoil write a single output to either ON or OFF in a
	// remote device and returns success or failed.
	WriteSingleCoil(slaveID byte, address uint16, isOn bool) error
	// WriteMultipleCoils forces each coil in a sequence of coils to either
	// ON or OFF in a remote device and returns success or failed.
	WriteMultipleCoils(slaveID byte, address, quantity uint16, value []byte) error

	// 16-bits

	// ReadInputRegistersBytes reads from 1 to 125 contiguous input registers in
	// a remote device and returns input registers.
	ReadInputRegistersBytes(slaveID byte, address, quantity uint16) (results []byte, err error)
	// ReadInputRegisters reads from 1 to 125 contiguous input registers in
	// a remote device and returns input registers.
	ReadInputRegisters(slaveID byte, address, quantity uint16) (results []uint16, err error)

	// ReadHoldingRegistersBytes reads the contents of a contiguous block of
	// holding registers in a remote device and returns register value.
	ReadHoldingRegistersBytes(slaveID byte, address, quantity uint16) (results []byte, err error)
	// ReadHoldingRegisters reads the contents of a contiguous block of
	// holding registers in a remote device and returns register value.
	ReadHoldingRegisters(slaveID byte, address, quantity uint16) (results []uint16, err error)

	// WriteSingleRegister writes a single holding register in a remote
	// device and returns success or failed.
	WriteSingleRegister(slaveID byte, address, value uint16) error
	// WriteMultipleRegistersBytes writes a block of contiguous registers
	// (1 to 123 registers) in a remote device and returns success or failed.
	WriteMultipleRegistersBytes(slaveID byte, address, quantity uint16, value []byte) error
	// WriteMultipleRegisters writes a block of contiguous registers
	// (1 to 123 registers) in a remote device and returns success or failed.
	WriteMultipleRegisters(slaveID byte, address, quantity uint16, value []uint16) error

	// ReadWriteMultipleRegistersBytes performs a combination of one read
	// operation and one write operation. It returns read registers value.
	ReadWriteMultipleRegistersBytes(slaveID byte, readAddress, readQuantity,
		writeAddress, writeQuantity uint16, value []byte) (results []byte, err error)
	// ReadWriteMultipleRegisters performs a combination of one read
	// operation and one write operation. It returns read registers value.
	ReadWriteMultipleRegisters(slaveID byte, readAddress, readQuantity,
		writeAddress, writeQuantity uint16, value []byte) (results []uint16, err error)

	// MaskWriteRegister modify the contents of a specified holding
	// register using a combination of an AND mask, an OR mask, and the
	// register's current contents. The function returns success or failed.
	MaskWriteRegister(slaveID byte, address, andMask, orMask uint16) error
	// ReadFIFOQueue reads the contents of a First-In-First-Out (FIFO) queue
	// of register in a remote device and returns FIFO value register.
	ReadFIFOQueue(slaveID byte, address uint16) (results []byte, err error)
}
