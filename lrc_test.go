package modbus

import (
	"testing"
)

func Test_lrc(t *testing.T) {
	var lrc lrc
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
			if got := lrc.reset().push(tt.args.bs...).value(); got != tt.want {
				t.Errorf("lrc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_lrc(b *testing.B) {
	var lrc lrc
	for i := 0; i < b.N; i++ {
		lrc.reset().push([]byte{0x02, 0x07, 0x01, 0x03, 0x01, 0x0a}...).value()
	}
}
