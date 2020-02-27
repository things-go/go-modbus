package main

import (
	"time"

	modbus "github.com/thinkgos/gomodbus"
	"github.com/thinkgos/gomodbus/mb"
)

func main() {
	p := modbus.NewRTUClientProvider()
	p.Address = "/dev/ttyUSB0"
	p.BaudRate = 115200
	p.DataBits = 8
	p.Parity = "N"
	p.StopBits = 1
	client := mb.NewClient(p)
	client.LogMode(true)
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

	select {}
}
