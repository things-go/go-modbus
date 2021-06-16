package main

import (
	"fmt"
	"time"

	modbus "github.com/things-go/go-modbus"
)

func main() {
	p := modbus.NewTCPClientProvider("localhost:502", modbus.WithEnableLogger())
	client := modbus.NewClient(p)
	err := client.Connect()
	if err != nil {
		fmt.Println("connect failed, ", err)
		return
	}
	defer client.Close()

	fmt.Println("starting")
	for {
		_, err := client.ReadCoils(1, 0, 10)
		if err != nil {
			fmt.Println(err.Error())
		}

		//	fmt.Printf("ReadDiscreteInputs %#v\r\n", results)

		time.Sleep(time.Second * 2)
	}
}
