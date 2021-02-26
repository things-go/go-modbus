package main

import (
	modbus "github.com/thinkgos/gomodbus/v2"
)

func main() {
	srv := modbus.NewTCPServer()
	srv.LogMode(true)
	srv.AddNodes(
		modbus.NewNodeRegister(
			1,
			0, 10, 0, 10,
			0, 10, 0, 10),
		modbus.NewNodeRegister(
			2,
			0, 10, 0, 10,
			0, 10, 0, 10),
		modbus.NewNodeRegister(
			3,
			0, 10, 0, 10,
			0, 10, 0, 10))

	err := srv.ListenAndServe(":502")
	if err != nil {
		panic(err)
	}
}
