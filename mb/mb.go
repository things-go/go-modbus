package mb

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	modbus "github.com/thinkgos/gomodbus"
	"github.com/thinkgos/timing"
)

// Handler 处理函数
type Handler interface {
	ProcReadCoils(address, quality uint16, valBuf []byte)
	ProcReadDiscretes(address, quality uint16, valBuf []byte)
	ProcReadHoldingRegisters(address, quality uint16, valBuf []byte)
	ProcReadInputRegisters(address, quality uint16, valBuf []byte)
	ProcResult(err error, result *Result)
}

const (
	// DefaultRandValue 单位ms
	// 默认随机值上限,它影响当超时请求入ready队列时,
	// 当队列满,会启动一个随机时间rand.Intn(v)*1ms 延迟入队
	// 用于需要重试的延迟重试时间
	DefaultRandValue = 50
	// DefaultReadyQueuesLength 默认就绪列表长度
	DefaultReadyQueuesLength = 128
)

// Client 客户端
type Client struct {
	modbus.Client
	randValue   int
	readyLength int
	ready       chan *Request
	handler     Handler
	panicHandle func(err interface{})
	ctx         context.Context
	cancel      context.CancelFunc
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
	Retry    byte          // 失败重试次数
	retryCnt byte          // 重试计数
	txCnt    uint64        // 发送计数
	errCnt   uint64        // 发送错误计数
	tm       timing.Timer  // 时间句柄
}

// NewClient 创建新的client
func NewClient(p modbus.ClientProvider, opts ...Option) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		Client:      modbus.NewClient(p),
		readyLength: DefaultReadyQueuesLength,
		handler:     &nopProc{},
		panicHandle: func(interface{}) {},
		ctx:         ctx,
		cancel:      cancel,
	}

	for _, f := range opts {
		f(c)
	}
	c.ready = make(chan *Request, c.readyLength)
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
		}
		req.tm = timing.AddOneShotJobFunc(func() {
			select {
			case <-sf.ctx.Done():
				return
			case sf.ready <- req:
			default:
				timing.Start(req.tm, time.Duration(rand.Intn(sf.randValue))*time.Millisecond)
			}
		}, req.ScanRate)

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
			return
		case req = <-sf.ready: // 查看是否有准备好的请求
			sf.procRequest(req)
		}
	}
}

func (sf *Client) procRequest(curReq *Request) {
	var err error
	var result []byte

	defer func() {
		if err := recover(); err != nil {
			sf.panicHandle(err)
		}
	}()

	curReq.txCnt++
	switch curReq.FuncCode {
	// Bit access read
	case modbus.FuncCodeReadCoils:
		result, err = sf.ReadCoils(curReq.SlaveID, curReq.Address, curReq.Quantity)
		if err != nil {
			curReq.errCnt++
		} else {
			sf.handler.ProcReadCoils(curReq.Address, curReq.Quantity, result)
		}
	case modbus.FuncCodeReadDiscreteInputs:
		result, err = sf.ReadDiscreteInputs(curReq.SlaveID, curReq.Address, curReq.Quantity)
		if err != nil {
			curReq.errCnt++
		} else {
			sf.handler.ProcReadDiscretes(curReq.Address, curReq.Quantity, result)
		}

	// 16-bit access read
	case modbus.FuncCodeReadHoldingRegisters:
		result, err = sf.ReadHoldingRegistersBytes(curReq.SlaveID, curReq.Address, curReq.Quantity)
		if err != nil {
			curReq.errCnt++
		} else {
			sf.handler.ProcReadHoldingRegisters(curReq.Address, curReq.Quantity, result)
		}

	case modbus.FuncCodeReadInputRegisters:
		result, err = sf.ReadInputRegistersBytes(curReq.SlaveID, curReq.Address, curReq.Quantity)
		if err != nil {
			curReq.errCnt++
		} else {
			sf.handler.ProcReadInputRegisters(curReq.Address, curReq.Quantity, result)
		}

		// FIFO read
		//case modbus.FuncCodeReadFIFOQueue:
		//	_, err = sf.ReadFIFOQueue(curReq.SlaveID, curReq.Address)
		//	if err != nil {
		//		curReq.errCnt++
		//	}
	}
	if err != nil && curReq.Retry > 0 {
		if curReq.retryCnt++; curReq.retryCnt < curReq.Retry {
			timing.Start(curReq.tm, time.Duration(rand.Intn(sf.randValue))*time.Millisecond) //rand.Intn(10)
		} else if curReq.ScanRate > 0 {
			timing.Start(curReq.tm)
		}
	} else if curReq.ScanRate > 0 {
		timing.Start(curReq.tm)
	}

	sf.handler.ProcResult(err, &Result{
		curReq.SlaveID,
		curReq.FuncCode,
		curReq.Address,
		curReq.Quantity,
		curReq.ScanRate,
		curReq.txCnt,
		curReq.errCnt,
	})
}

type nopProc struct{}

func (nopProc) ProcReadCoils(uint16, uint16, []byte)            {}
func (nopProc) ProcReadDiscretes(uint16, uint16, []byte)        {}
func (nopProc) ProcReadHoldingRegisters(uint16, uint16, []byte) {}
func (nopProc) ProcReadInputRegisters(uint16, uint16, []byte)   {}
func (nopProc) ProcResult(_ error, result *Result) {
	log.Printf("Tx=%d,Err=%d,SlaveID=%d,FC=%d,Address=%d,Quantity=%d,SR=%dms",
		result.TxCnt, result.ErrCnt, result.SlaveID, result.FuncCode,
		result.Address, result.Quantity, result.ScanRate/time.Millisecond)
}
