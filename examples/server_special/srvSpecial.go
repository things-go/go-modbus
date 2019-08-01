package main

import (
	"log"
	"net/http"
	"time"

	modbus "github.com/thinkgos/gomodbus"
)

func main() {
	srv := modbus.NewTCPServerSpecial()
	if err := srv.AddRemoteServer("192.168.199.148:2404"); err != nil {
		panic(err)
	}
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

	srv.SetOnConnectHandler(func(c modbus.TCPServerSpecial) error {
		_, err := c.UnderlyingConn().Write([]byte("hello world"))
		return err
	})

	srv.SetConnectionLostHandler(func(c modbus.TCPServerSpecial) {
		log.Println("connect lost")
	})

	if err := srv.Start(); err != nil {
		panic(err)
	}
	go func() {
		time.Sleep(time.Second * 10)
		srv.Close()
	}()
	if err := http.ListenAndServe(":6060", nil); err != nil {
		panic(err)
	}
}
