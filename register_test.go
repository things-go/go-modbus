package modbus

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_getBits(t *testing.T) {
	type args struct {
		buf   []byte
		start uint16
		nBits uint16
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{"获取0-8位,共8个", args{[]byte{0xaa, 0x5}, 0, 8}, 0xaa},
		{"获取0-4位,共4个", args{[]byte{0xaa, 0x55}, 0, 4}, 0x0a},
		{"获取4-8位,共4个", args{[]byte{0xaa, 0x55}, 4, 4}, 0x0a},
		{"获取4-12位,共4个", args{[]byte{0xaa, 0x55}, 4, 8}, 0x5a},
		{"获取7-9位,共3个", args{[]byte{0xaa, 0x55}, 7, 3}, 0x03},
		{"获取9-16位,共7个", args{[]byte{0xaa, 0x55}, 9, 7}, 0x2a},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBits(tt.args.buf, tt.args.start, tt.args.nBits); got != tt.want {
				t.Errorf("getBits() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_setBits(t *testing.T) {
	type args struct {
		buf   []byte
		start uint16
		nBits uint16
		value byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"设置0-8位,共8个", args{[]byte{0x00, 0x00}, 0, 8, 0xaa}, []byte{0xaa, 0x00}},
		{"设置0-4位,共4个", args{[]byte{0x00, 0x00}, 0, 4, 0x0a}, []byte{0x0a, 0x00}},
		{"设置4-12位,共8个", args{[]byte{0x00, 0x00}, 4, 8, 0xaa}, []byte{0xa0, 0x0a}},
		{"设置7-9位,共3个", args{[]byte{0x00, 0x00}, 7, 3, 0x07}, []byte{0x80, 0x03}},
		{"设置1位,共1个", args{[]byte{0x00, 0x00}, 1, 1, 0x01}, []byte{0x02, 0x00}},
		{"设置9-16位,共7个", args{[]byte{0x00, 0x00}, 9, 7, 0x7f}, []byte{0x00, 0xfe}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBits(tt.args.buf, tt.args.start, tt.args.nBits, tt.args.value)
			if !bytes.Equal(tt.args.buf, tt.want) {
				t.Errorf("setBits() = %#v, want %#v", tt.args.buf, tt.want)
			}
		})
	}
}

func Benchmark_getBits(b *testing.B) {
	val := []byte{0x00, 0x02, 0x03, 0x04, 0x05}
	for i := 0; i < b.N; i++ {
		getBits(val, 1, 30)
	}
}

func Benchmark_setBits(b *testing.B) {
	val := []byte{0x00, 0x02, 0x03, 0x04, 0x05}
	for i := 0; i < b.N; i++ {
		setBits(val, 12, 8, 0xaa)
	}
}

func TestNodeRegister_SlaveID(t *testing.T) {
	tests := []struct {
		name string
		this *NodeRegister
		want uint8
	}{
		{
			"slave ID same",
			&NodeRegister{slaveID: 0x01},
			0x01,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.this.SlaveID(); got != tt.want {
				t.Errorf("NodeRegister.SlaveID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_WriteCoils(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
		valBuf  []byte
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteCoils(tt.args.address, tt.args.quality, tt.args.valBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteCoils() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_ReadCoils(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.ReadCoils(tt.args.address, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.ReadCoils() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeRegister.ReadCoils() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_WriteDiscretes(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
		valBuf  []byte
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteDiscretes(tt.args.address, tt.args.quality, tt.args.valBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteDiscretes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_ReadDiscretes(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.ReadDiscretes(tt.args.address, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.ReadDiscretes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeRegister.ReadDiscretes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_WriteHoldingsBytes(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
		valBuf  []byte
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteHoldingsBytes(tt.args.address, tt.args.quality, tt.args.valBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteHoldingsBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_WriteHoldings(t *testing.T) {
	type args struct {
		address uint16
		valBuf  []uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteHoldings(tt.args.address, tt.args.valBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteHoldings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_ReadHoldings(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		want    []uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.ReadHoldings(tt.args.address, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.ReadHoldings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeRegister.ReadHoldings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_WriteInputsBytes(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
		regBuf  []byte
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteInputsBytes(tt.args.address, tt.args.quality, tt.args.regBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteInputsBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_WriteInputs(t *testing.T) {
	type args struct {
		address uint16
		valBuf  []uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.WriteInputs(tt.args.address, tt.args.valBuf); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.WriteInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeRegister_ReadInputsBytes(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.ReadInputsBytes(tt.args.address, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.ReadInputsBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeRegister.ReadInputsBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_ReadInputs(t *testing.T) {
	type args struct {
		address uint16
		quality uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		want    []uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.ReadInputs(tt.args.address, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.ReadInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeRegister.ReadInputs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeRegister_MaskWriteHolding(t *testing.T) {
	type args struct {
		address uint16
		andMask uint16
		orMask  uint16
	}
	tests := []struct {
		name    string
		this    *NodeRegister
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.MaskWriteHolding(tt.args.address, tt.args.andMask, tt.args.orMask); (err != nil) != tt.wantErr {
				t.Errorf("NodeRegister.MaskWriteHolding() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
