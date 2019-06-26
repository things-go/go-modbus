[![GoDoc](https://godoc.org/github.com/thinkgos/gomodbus?status.svg)](https://godoc.org/github.com/thinkgos/gomodbus)[!
[Build Status](https://www.travis-ci.org/thinkgos/gomodbus.svg?branch=master)](https://www.travis-ci.org/thinkgos/gomodbus)
[![codecov](https://codecov.io/gh/thinkgos/gomodbus/branch/master/graph/badge.svg)](https://codecov.io/gh/thinkgos/gomodbus)
### go modbus Supported formats
- modbus TCP Client
- modbus Serial(RTU,ASCII) Client
- modbus TCP Server

### 特性
- 临时对象缓冲池,减少内存分配
- 快速编码,解码
- interface设计,提供扩展性
- 简单的丰富的API

大量参考了!为了用于生产环境[goburrow](https://github.com/goburrow/modbus)

### Supported functions

-------------------
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
----------
```
    // modbus TCP Client
    p := modbus.NewTCPClientProvider(":502")
	p.Logger = log.New(os.Stdout, "", log.LstdFlags)
	client := modbus.NewClient(p)
	client.LogMode(true)
	err := client.Connect()
	if err != nil {
		fmt.Println("connect", err)
		return
	}
	defer client.Close()
	for {
		_, err := client.ReadCoils(1, 0, 10)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("ReadDiscreteInputs %#v\r\n", results)
		}
		time.Sleep(time.Second * 5)
	}
```

```
    // modbus RTU/ASCII Client
    p := modbus.NewTCPClientProvider("COM1")
    p.BaudRate = 115200
	p.DataBits = 8
	p.Parity = "N"
	p.StopBits = 1
	p.Logger = log.New(os.Stdout, "", log.LstdFlags)
	client := modbus.NewClient(p)
	client.LogMode(true)
	err := client.Connect()
	if err != nil {
		fmt.Println("connect", err)
		return
	}
	defer client.Close()
	for {
		_, err := client.ReadCoils(1, 0, 10)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("ReadDiscreteInputs %#v\r\n", results)
		}
		time.Sleep(time.Second * 5)
	}
```
```
    // modbus TCP Server
	srv := modbus.NewTCPServer(":502")
	srv.Logger = log.New(os.Stdout, "modbus", log.Ltime)
	srv.LogMode(true)
	srv.AddNode(modbus.NewNodeRegister(
		1,
		0, 10, []byte{0xfa, 0xa0},
		0, 10, []byte{0xa5, 0x0a},
		0, []uint16{0x1234, 0x4567, 0x1234, 0x4567, 0x1234, 0x4567, 0x4567, 0x1234, 0x4567, 0x1234},
		0, []uint16{0x4567, 0x1234, 0x4567, 0x1234, 0x4567, 0x1234, 0x4567, 0x1234, 0x4567, 0x1234},
	))
	srv.ServerModbus()
```
### References
----------
-   [Modbus Specifications and Implementation Guides](http://www.modbus.org/specs.php)
