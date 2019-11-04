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
			New: func() interface{} {
				return &protocolFrame{make([]byte, 0, size)}
			},
		},
	}
}

func (sf *pool) get() *protocolFrame {
	v := sf.pl.Get().(*protocolFrame)
	v.adu = v.adu[:0]
	return v
}

func (sf *pool) put(buffer *protocolFrame) {
	sf.pl.Put(buffer)
}
