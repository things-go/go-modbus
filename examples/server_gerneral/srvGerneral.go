package main

import (
	"net/http"

	modbus "github.com/thinkgos/gomodbus"

	_ "net/http/pprof"
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
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			panic(err)
		}
	}()

	err := srv.ListenAndServe(":502")
	if err != nil {
		panic(err)
	}
}
