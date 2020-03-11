package mb

// Handler 处理函数
type Handler interface {
	ProcReadCoils(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadDiscretes(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadHoldingRegisters(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadInputRegisters(slaveID byte, address, quality uint16, valBuf []byte)
	ProcResult(err error, result *Result)
}

// NopProc implement interface Handler
type NopProc struct{}

// ProcReadCoils implement interface Handler
func (NopProc) ProcReadCoils(byte, uint16, uint16, []byte) {}

// ProcReadDiscretes implement interface Handler
func (NopProc) ProcReadDiscretes(byte, uint16, uint16, []byte) {}

// ProcReadHoldingRegisters implement interface Handler
func (NopProc) ProcReadHoldingRegisters(byte, uint16, uint16, []byte) {}

// ProcReadInputRegisters implement interface Handler
func (NopProc) ProcReadInputRegisters(byte, uint16, uint16, []byte) {}

// ProcResult implement interface Handler
func (NopProc) ProcResult(error, *Result) {
	//log.Printf("Tx=%d,Err=%d,SlaveID=%d,FC=%d,Address=%d,Quantity=%d,SR=%dms",
	//	result.TxCnt, result.ErrCnt, result.SlaveID, result.FuncCode,
	//	result.Address, result.Quantity, result.ScanRate/time.Millisecond)
}
