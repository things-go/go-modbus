package mb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	modbus "github.com/thinkgos/gomodbus/v2"
	"github.com/thinkgos/timing/v4"
)

const (
	// DefaultRandValue 单位ms
	// 默认随机值上限,它影响当超时请求入ready队列时,
	// 当队列满,会启动一个随机时间rand.Intn(v)*1ms 延迟入队
	// 用于需要重试的延迟重试时间
	DefaultRandValue = 50
	// DefaultReadyQueuesLength 默认就绪列表长度
	DefaultReadyQueuesLength = 256
)

// Client 客户端
type Client struct {
	modbus.Client
	randValue      int
	readyQueueSize int
	ready          chan *Request
	handler        Handler
	panicHandle    func(err interface{})
	ctx            context.Context
	cancel         context.CancelFunc
}

// Result 某个请求的结果与参数
type Result struct {
	SlaveID  byte          // 从机地址
	FuncCode byte          // 功能码
	Address  uint16        // 请求数据用实际地址
	Quantity uint16        // 请求数量
	ScanRate time.Duration // 扫描速率scan rate
	TxCnt    uint64        // 发送计数
	ErrCnt   uint64        // 发送错误计数
}

// Request 请求
type Request struct {
	SlaveID  byte          // 从机地址
	FuncCode byte          // 功能码
	Address  uint16        // 请求数据用实际地址
	Quantity uint16        // 请求数量
	ScanRate time.Duration // 扫描速率scan rate
	txCnt    uint64        // 发送计数
	errCnt   uint64        // 发送错误计数
	tm       *timing.Timer
}

// New 创建新的client
func New(p modbus.ClientProvider, opts ...Option) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		Client:         modbus.NewClient(p),
		randValue:      DefaultRandValue,
		readyQueueSize: DefaultReadyQueuesLength,
		handler:        &NopProc{},
		panicHandle:    func(interface{}) {},
		ctx:            ctx,
		cancel:         cancel,
	}

	for _, opt := range opts {
		opt(c)
	}
	c.ready = make(chan *Request, c.readyQueueSize)
	return c
}

// Start 启动
func (sf *Client) Start() error {
	if err := sf.Connect(); err != nil {
		return err
	}
	go sf.readPoll()
	return nil
}

// Close 关闭
func (sf *Client) Close() error {
	sf.cancel()
	return sf.Client.Close()
}

// AddGatherJob 增加采集任务
func (sf *Client) AddGatherJob(r Request) error {
	var quantityMax int

	if err := sf.ctx.Err(); err != nil {
		return err
	}

	if r.SlaveID < modbus.AddressMin || r.SlaveID > modbus.AddressMax {
		return fmt.Errorf("modbus: slaveID '%v' must be between '%v' and '%v'",
			r.SlaveID, modbus.AddressMin, modbus.AddressMax)
	}

	switch r.FuncCode {
	case modbus.FuncCodeReadCoils, modbus.FuncCodeReadDiscreteInputs:
		quantityMax = modbus.ReadBitsQuantityMax
	case modbus.FuncCodeReadInputRegisters, modbus.FuncCodeReadHoldingRegisters:
		quantityMax = modbus.ReadRegQuantityMax
	default:
		return errors.New("invalid function code")
	}

	address := r.Address
	remain := int(r.Quantity)
	for remain > 0 {
		count := remain
		if count > quantityMax {
			count = quantityMax
		}

		req := &Request{
			SlaveID:  r.SlaveID,
			FuncCode: r.FuncCode,
			Address:  address,
			Quantity: uint16(count),
			ScanRate: r.ScanRate,
			tm:       timing.NewTimer(),
		}
		req.tm.WithJobFunc(func() {
			select {
			case <-sf.ctx.Done():
				return
			case sf.ready <- req:
			default:
				timing.Add(req.tm, time.Duration(rand.Intn(sf.randValue))*time.Millisecond)
			}
		})
		timing.Add(req.tm, req.ScanRate)

		address += uint16(count)
		remain -= count
	}
	return nil
}

// 读协程
func (sf *Client) readPoll() {
	var req *Request

	for {
		select {
		case <-sf.ctx.Done():
			log.Println("read pool exit!")
			return
		case req = <-sf.ready: // 查看是否有准备好的请求
			sf.procRequest(req)
		}
	}
}

func (sf *Client) procRequest(req *Request) {
	var err error
	var result []byte

	defer func() {
		if err := recover(); err != nil {
			sf.panicHandle(err)
		}
	}()

	req.txCnt++
	switch req.FuncCode {
	// Bit access read
	case modbus.FuncCodeReadCoils:
		result, err = sf.ReadCoils(req.SlaveID, req.Address, req.Quantity)
		if err == nil {
			sf.handler.ProcReadCoils(req.SlaveID, req.Address, req.Quantity, result)
		}
	case modbus.FuncCodeReadDiscreteInputs:
		result, err = sf.ReadDiscreteInputs(req.SlaveID, req.Address, req.Quantity)
		if err == nil {
			sf.handler.ProcReadDiscretes(req.SlaveID, req.Address, req.Quantity, result)
		}

	// 16-bit access read
	case modbus.FuncCodeReadHoldingRegisters:
		result, err = sf.ReadHoldingRegistersBytes(req.SlaveID, req.Address, req.Quantity)
		if err == nil {
			sf.handler.ProcReadHoldingRegisters(req.SlaveID, req.Address, req.Quantity, result)
		}

	case modbus.FuncCodeReadInputRegisters:
		result, err = sf.ReadInputRegistersBytes(req.SlaveID, req.Address, req.Quantity)
		if err == nil {
			sf.handler.ProcReadInputRegisters(req.SlaveID, req.Address, req.Quantity, result)
		}

		// FIFO read
		//case modbus.FuncCodeReadFIFOQueue:
		//	_, err = sf.ReadFIFOQueue(req.SlaveID, req.Address)
		//	if err != nil {
		//		req.errCnt++
		//	}
	}
	if err != nil {
		req.errCnt++
	}

	if req.ScanRate > 0 {
		timing.Add(req.tm, req.ScanRate)
	}
	sf.handler.ProcResult(err, &Result{
		req.SlaveID,
		req.FuncCode,
		req.Address,
		req.Quantity,
		req.ScanRate,
		req.txCnt,
		req.errCnt,
	})
}
