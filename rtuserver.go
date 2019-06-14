package modbus

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/goburrow/serial"
	"github.com/thinkgos/library/elog"
)

// TODO: BUG can't work

// RTUServer modbus rtu server
type RTUServer struct {
	config  *serial.Config
	nodeReg *NodeRegister
	*serverHandler
	logs
}

// NewRTUServer 创建一个rtu server
func NewRTUServer(c *serial.Config) *RTUServer {
	return &RTUServer{
		serverHandler: newServerHandler(),
		config:        c,
		logs: logs{
			Elog: elog.NewElog(nil),
		},
	}
}

// SetNodeRegister 设置从机地址和寄存器值
func (this *RTUServer) SetNodeRegister(nodeReg *NodeRegister) {
	this.nodeReg = nodeReg
}

// GetSlaveID 获得从机地址
func (this *RTUServer) GetSlaveID() byte {
	return this.nodeReg.slaveID
}

// ServerModbus 服务
func (this *RTUServer) ServerModbus() {
	port, err := serial.Open(this.config)
	if err != nil {
		this.Error("failed to open %s: %v\n", this.config.Address, err)
		return
	}
	defer port.Close()
	this.Debug("modbus TCP server running")
	frame := &protocolRTUFrame{}
	readbuf := make([]byte, 512)
	for {
		bytesRead, err := port.Read(readbuf)
		if err != nil {
			if err != io.EOF {
				this.Error("read error %v\n", err)
				return
			}
			// cnt >0 do nothing
			// cnt == continue next do it
		}
		if bytesRead == 0 {
			continue
		}

		response, err := this.frameHandler(frame, readbuf[:bytesRead])
		if err != nil {
			this.Error("frameHandler: %v", err)
			continue
		}
		_, err = port.Write(response)
		if err != nil {
			this.Error("write: %v", err)
		}
	}
}

// modbus 包处理
func (this *RTUServer) frameHandler(frame *protocolRTUFrame, packet []byte) ([]byte, error) {
	var data []byte

	fra, err := newRTUFrame(packet)
	if err != nil {
		return nil, fmt.Errorf("bad packet error %v", err)
	}
	this.Debug("request raw frame: % x", packet)
	if fra.slaveID != this.nodeReg.slaveID || fra.slaveID != addressBroadCast {
		return nil, fmt.Errorf("packet not for me %d", fra.slaveID)
	}
	if handle, ok := this.function[fra.funcCode]; ok {
		data, err = handle(this.nodeReg, fra.data)
	} else {
		err = &ExceptionError{ExceptionCodeIllegalFunction}
	}
	if fra.slaveID == addressBroadCast {
		return nil, fmt.Errorf("broadcast address,not need response")
	}

	response := fra.copy()
	if err != nil {
		response.funcCode |= 0x80
		data = []byte{err.(*ExceptionError).ExceptionCode}
	}
	response.data = data
	rsp := response.bytes()
	this.Debug("response raw frame: % x", rsp)
	return rsp, nil
}

// RTUFrame is the Modbus TCP frame.
type rtuFrame struct {
	slaveID  uint8
	funcCode uint8
	data     []byte
	crc      uint16
}

// NewRTUFrame converts a packet to a Modbus TCP frame.
func newRTUFrame(packet []byte) (*rtuFrame, error) {
	if len(packet) < 5 { // Check the that the packet length.
		return nil, fmt.Errorf("RTU Frame error: packet less than 5 bytes: %v", packet)
	}
	crc := crc16(packet[:len(packet)-2])
	expect := binary.LittleEndian.Uint16(packet[len(packet)-2:])
	if crc != expect { // Check the CRC.
		return nil, fmt.Errorf("RTU Frame error: CRC (expected 0x%x, got 0x%x)", expect, crc)
	}

	return &rtuFrame{
		slaveID:  uint8(packet[0]),
		funcCode: uint8(packet[1]),
		data:     packet[2 : len(packet)-2], // pass slaveID funcCode and crc
		crc:      crc,
	}, nil
}

func (this *rtuFrame) copy() *rtuFrame {
	return this
}

func (this *rtuFrame) bytes() []byte {
	b := make([]byte, len(this.data)+4)
	b[0] = this.slaveID
	b[1] = this.funcCode
	copy(b[2:], this.data)
	crc := crc16(b[0 : len(b)-2])
	binary.LittleEndian.PutUint16(b[len(b)-2:], crc)
	return b
}
