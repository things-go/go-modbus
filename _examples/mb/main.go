package main

import (
	"log"
	"time"

	"github.com/goburrow/serial"
	modbus "github.com/thinkgos/gomodbus/v2"
	"github.com/thinkgos/gomodbus/v2/mb"
)

func main() {
	p := modbus.NewRTUClientProvider(modbus.WithEnableLogger(),
		modbus.WithSerialConfig(serial.Config{
			Address:  "/dev/ttyUSB0",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "N",
			Timeout:  modbus.SerialDefaultTimeout,
		}))

	client := mb.New(p, mb.WitchHandler(&handler{}))
	err := client.Start()
	if err != nil {
		panic(err)
	}

	err = client.AddGatherJob(mb.Request{
		SlaveID:  1,
		FuncCode: modbus.FuncCodeReadHoldingRegisters,
		Address:  0,
		Quantity: 300,
		ScanRate: time.Second,
	})

	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Second * 10)
	}
}

type handler struct {
	mb.NopProc
}

func (handler) ProcResult(_ error, result *mb.Result) {
	log.Printf("Tx=%d,Err=%d,SlaveID=%d,FC=%d,Address=%d,Quantity=%d,SR=%dms",
		result.TxCnt, result.ErrCnt, result.SlaveID, result.FuncCode,
		result.Address, result.Quantity, result.ScanRate/time.Millisecond)
}
