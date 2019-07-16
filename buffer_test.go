package modbus

import (
	"testing"
)

func Test_pool(t *testing.T) {
	p := newPool(tcpAduMaxSize)
	frame := p.get()
	if len(frame.adu) != 0 {
		t.Errorf("pool.get() got len = %v, want %v", len(frame.adu), 0)
	}
	if cap(frame.adu) != tcpAduMaxSize {
		t.Errorf("pool.get() got cap = %v, want %v", cap(frame.adu), tcpAduMaxSize)
	}

	p = newPool(asciiCharacterMaxSize)
	frame = p.get()
	if len(frame.adu) != 0 {
		t.Errorf("pool.get() got len = %v, want %v", len(frame.adu), 0)
	}
	if cap(frame.adu) != asciiCharacterMaxSize {
		t.Errorf("pool.get() got cap = %v, want %v", cap(frame.adu), asciiCharacterMaxSize)
	}
}
