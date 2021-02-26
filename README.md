# go modbus

modbus write in pure go, support rtu,ascii,tcp master library,also support tcp slave.

[![GoDoc](https://godoc.org/github.com/thinkgos/gomodbus?status.svg)](https://godoc.org/github.com/thinkgos/gomodbus)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/thinkgos/gomodbus/v2?tab=doc)
[![Build Status](https://www.travis-ci.org/thinkgos/gomodbus.svg?branch=master)](https://www.travis-ci.org/thinkgos/gomodbus)
[![codecov](https://codecov.io/gh/thinkgos/gomodbus/branch/master/graph/badge.svg)](https://codecov.io/gh/thinkgos/gomodbus)
![Action Status](https://github.com/thinkgos/gomodbus/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thinkgos/gomodbus)](https://goreportcard.com/report/github.com/thinkgos/gomodbus)
[![Licence](https://img.shields.io/github/license/thinkgos/gomodbus)](https://raw.githubusercontent.com/thinkgos/gomodbus/master/LICENSE)
[![Tag](https://img.shields.io/github/v/tag/thinkgos/gomodbus)](https://github.com/thinkgos/gomodbus/tags)
[![Sourcegraph](https://sourcegraph.com/github.com/thinkgos/gomodbus/-/badge.svg)](https://sourcegraph.com/github.com/thinkgos/gomodbus?badge)


### Supported formats

- modbus Serial(RTU,ASCII) Client
- modbus TCP Client
- modbus TCP Server

### Features

- object pool design,reduce memory allocation
- fast encode and decode
- interface design
- simple API and support raw data api

### Installation

Use go get.
```bash
    go get github.com/thinkgos/gomodbus/v2
```

Then import the modbus package into your own code.
```bash
    import modbus "github.com/thinkgos/gomodbus/v2"
```

### Supported functions

---

Bit access:
*   Read Discrete Inputs
*   Read Coils
*   Write Single Coil
*   Write Multiple Coils

16-bit access:
*   Read Input Registers
*   Read Holding Registers
*   Write Single Register
*   Write Multiple Registers
*   Read/Write Multiple Registers
*   Mask Write Register
*   Read FIFO Queue

### Example

---


modbus RTU/ASCII client see [example](_examples/client_rtu_ascii)

[embedmd]:# (_examples/client_rtu_ascii/main.go go)
```go
package main

import (
	"fmt"
	"time"

	"github.com/goburrow/serial"
	modbus "github.com/thinkgos/gomodbus/v2"
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

	client := modbus.NewClient(p)
	err := client.Connect()
	if err != nil {
		fmt.Println("connect failed, ", err)
		return
	}
	defer client.Close()

	fmt.Println("starting")
	for {
		_, err := client.ReadCoils(3, 0, 10)
		if err != nil {
			fmt.Println(err.Error())
		}

		//	fmt.Printf("ReadDiscreteInputs %#v\r\n", results)

		time.Sleep(time.Second * 2)
	}
}
```


modbus TCP client see [example](_examples/client_tcp)

[embedmd]:# (_examples/client_tcp/main.go go)
```go
package main

import (
	"fmt"
	"time"

	modbus "github.com/thinkgos/gomodbus/v2"
)

func main() {
	p := modbus.NewTCPClientProvider("192.168.199.188:502", modbus.WithEnableLogger())
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
```

modbus TCP server see [example](_examples/server_tcp)

[embedmd]:# (_examples/server_tcp/main.go go)
```go
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
```

### References

---

- [Modbus Specifications and Implementation Guides](http://www.modbus.org/specs.php)
- [goburrow](https://github.com/goburrow/modbus)

## Donation

if package help you a lot,you can support us by:

**Alipay**

![alipay](https://github.com/thinkgos/thinkgos/blob/master/asserts/alipay.jpg)

**WeChat Pay**

![wxpay](https://github.com/thinkgos/thinkgos/blob/master/asserts/wxpay.jpg)
