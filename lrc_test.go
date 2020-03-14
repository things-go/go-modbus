package modbus

import (
	"testing"
)

func Test_LRC(t *testing.T) {
	var lrc LRC
	type args struct {
		bs []byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{"lrc校验", args{[]byte{0x01, 0x03, 0x01, 0x0a}}, 0xf1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lrc.Reset().Push(tt.args.bs...).Value(); got != tt.want {
				t.Errorf("lrc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_LRC(b *testing.B) {
	var lrc LRC
	for i := 0; i < b.N; i++ {
		lrc.Reset().Push([]byte{0x02, 0x07, 0x01, 0x03, 0x01, 0x0a}...).Value()
	}
}
