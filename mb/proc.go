package mb

// Handler 处理函数
type Handler interface {
	ProcReadCoils(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadDiscretes(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadHoldingRegisters(slaveID byte, address, quality uint16, valBuf []byte)
	ProcReadInputRegisters(slaveID byte, address, quality uint16, valBuf []byte)
	ProcResult(err error, result *Result)
}

type NopProc struct{}

func (NopProc) ProcReadCoils(byte, uint16, uint16, []byte)            {}
func (NopProc) ProcReadDiscretes(byte, uint16, uint16, []byte)        {}
func (NopProc) ProcReadHoldingRegisters(byte, uint16, uint16, []byte) {}
func (NopProc) ProcReadInputRegisters(byte, uint16, uint16, []byte)   {}
func (NopProc) ProcResult(_ error, result *Result) {
	//log.Printf("Tx=%d,Err=%d,SlaveID=%d,FC=%d,Address=%d,Quantity=%d,SR=%dms",
	//	result.TxCnt, result.ErrCnt, result.SlaveID, result.FuncCode,
	//	result.Address, result.Quantity, result.ScanRate/time.Millisecond)
}
