package modbus

import (
	"sync"
)

type pool struct {
	pl *sync.Pool
}

func newPool(size int) *pool {
	return &pool{
		&sync.Pool{
			New: func() interface{} { return &protocolFrame{make([]byte, 0, size)} },
		},
	}
}

func (sf *pool) get() *protocolFrame {
	return sf.pl.Get().(*protocolFrame)
}

func (sf *pool) put(buffer *protocolFrame) {
	buffer.adu = buffer.adu[:0]
	sf.pl.Put(buffer)
}
