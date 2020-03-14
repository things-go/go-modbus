package modbus

import (
	"testing"
)

func Test_CRC16(t *testing.T) {
	type args struct {
		bs []byte
	}
	tests := []struct {
		name string
		args args
		want uint16
	}{
		{"CRC16 ", args{[]byte{0x01, 0x02, 0x03, 0x04, 0x05}}, 0xbb2a},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CRC16(tt.args.bs); got != tt.want {
				t.Errorf("CRC16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_CRC16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CRC16([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	}
}
