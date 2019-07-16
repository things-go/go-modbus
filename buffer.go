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

func (this *pool) get() *protocolFrame {
	v := this.pl.Get().(*protocolFrame)
	v.adu = v.adu[:0]
	return v
}

func (this *pool) put(buffer *protocolFrame) {
	this.pl.Put(buffer)
}
