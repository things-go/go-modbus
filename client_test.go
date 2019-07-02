package modbus

import (
	"reflect"
	"testing"
)

func Test_pduDataBlock(t *testing.T) {
	type args struct {
		value []uint16
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"", args{[]uint16{0x1234, 0x5678}}, []byte{0x12, 0x34, 0x56, 0x78}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pduDataBlock(tt.args.value...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pduDataBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pduDataBlockSuffix(t *testing.T) {
	type args struct {
		suffix []byte
		value  []uint16
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"", args{[]byte{0x12, 0x34}, []uint16{0x4567}}, []byte{0x45, 0x67, 0x02, 0x12, 0x34}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pduDataBlockSuffix(tt.args.suffix, tt.args.value...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pduDataBlockSuffix() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_bytes2Uint16(t *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name string
		args args
		want []uint16
	}{
		{"byte to uint16", args{[]byte{0x12, 0x34}}, []uint16{0x1234}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytes2Uint16(tt.args.buf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bytes2Uint16() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Benchmark_dataBlock(b *testing.B) {
	data := []uint16{0x01, 0x10, 0x8A, 0x00, 0x00, 0x03, 0xAA, 0x10}
	for i := 0; i < b.N; i++ {
		pduDataBlock(data...)
	}
}

func Benchmark_dataBlockSuffix(b *testing.B) {
	suffix := []byte{0x01, 0x10, 0x8A, 0x00, 0x00, 0x03, 0xAA, 0x10}
	data := []uint16{0x01, 0x10, 0x8A, 0x00, 0x00, 0x03, 0xAA, 0x10}
	for i := 0; i < b.N; i++ {
		pduDataBlockSuffix(suffix, data...)
	}
}
