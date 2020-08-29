package modbus

import (
	"reflect"
	"testing"
)

func Test_newServerHandler(t *testing.T) {
	sh := newServerCommon()
	tests := []struct {
		name string
		got  *serverCommon
		want *serverCommon
	}{
		{"just cover", sh, sh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("newServerCommon() = %v, want %v", tt.want, tt.want)
			}
		})
	}
}

func Test_funcReadDiscreteInputs(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{readReg, []byte{0, 0, 0}}, nil, true},
		{"数量小于1或大于2000", args{readReg, []byte{0x00, 0x00, 0x07, 0xd1}}, nil, true},
		{"正常读0起始，8位", args{readReg, []byte{0x00, 0x00, 0x00, 0x08}}, []byte{0x01, 0xaa}, false},
		{"正常读4起始，9位", args{readReg, []byte{0x00, 0x04, 0x00, 0x09}}, []byte{0x02, 0x5a, 0x01}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcReadDiscreteInputs(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcReadDiscreteInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcReadDiscreteInputs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_funcReadCoils(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{readReg, []byte{0, 0, 0}}, nil, true},
		{"数量小于1或大于2000", args{readReg, []byte{0x00, 0x00, 0x07, 0xd1}}, nil, true},
		{"正常读0起始，8位", args{readReg, []byte{0x00, 0x00, 0x00, 0x08}}, []byte{0x01, 0x55}, false},
		{"正常读4起始，9位", args{readReg, []byte{0x00, 0x04, 0x00, 0x09}}, []byte{0x02, 0xa5, 0x00}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcReadCoils(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcReadCoils() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcReadCoils() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_funcWriteSingleCoil(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{newNodeReg(), []byte{0, 0, 0}}, nil, true},
		{"线圈值不是0x000或0xff00", args{newNodeReg(), []byte{0x00, 0x00, 0x01, 0x00}}, nil, true},
		{"写0xff00", args{newNodeReg(), []byte{0x00, 0x00, 0xff, 0x00}}, []byte{0x00, 0x00, 0xff, 0x00}, false},
		{"写0xff00", args{newNodeReg(), []byte{0x00, 0x00, 0x00, 0x00}}, []byte{0x00, 0x00, 0x00, 0x00}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcWriteSingleCoil(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcWriteSingleCoil() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcWriteSingleCoil() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_funcWriteMultiCoils(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncWriteMultiMinSize[5]", args{newNodeReg(), []byte{0, 0, 0}}, nil, true},
		{"数量小于1或大于2000", args{newNodeReg(), []byte{0x00, 0x00, 0x07, 0xd1, 0x00}}, nil, true},
		{"写值", args{newNodeReg(), []byte{0x00, 0x00, 0x00, 0x01, 0x01, 0x77}}, []byte{0x00, 0x00, 0x00, 0x01}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcWriteMultiCoils(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcWriteMultiCoils() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcWriteMultiCoils() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_funcReadInputRegisters(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{readReg, []byte{0x00, 0x00, 0x00}}, nil, true},
		{"数量小于1或大于125", args{readReg, []byte{0x00, 0x00, 0x00, 0x7e}}, nil, true},
		{"读值", args{readReg, []byte{0x00, 0x00, 0x00, 0x1}}, []byte{0x02, 0x90, 0x12}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcReadInputRegisters(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcReadInputRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcReadInputRegisters() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_funcReadHoldingRegisters(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{readReg, []byte{0x00, 0x00, 0x00}}, nil, true},
		{"数量小于1或大于125", args{readReg, []byte{0x00, 0x00, 0x00, 0x7e}}, nil, true},
		{"读值", args{readReg, []byte{0x00, 0x00, 0x00, 0x1}}, []byte{0x02, 0x12, 0x34}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcReadHoldingRegisters(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcReadHoldingRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcReadHoldingRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_funcWriteSingleRegister(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncReadMinSize[4]", args{newNodeReg(), []byte{0x00, 0x00, 0x00}}, nil, true},
		{"写值", args{newNodeReg(), []byte{0x00, 0x00, 0x00, 0x1}}, []byte{0x00, 0x00, 0x00, 0x1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcWriteSingleRegister(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcWriteSingleRegister() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcWriteSingleRegister() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_funcWriteMultiHoldingRegisters(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncWriteMultiMinSize[5]", args{newNodeReg(), []byte{0x00, 0x00, 0x00}}, nil, true},
		{"数量小于1或大于123", args{newNodeReg(), []byte{0x00, 0x00, 0x00, 0x7c, 0x01}}, nil, true},
		{"写值", args{newNodeReg(), []byte{0x00, 0x00, 0x00, 0x01, 0x02, 0x12, 0x34}}, []byte{0x00, 0x00, 0x00, 0x01}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcWriteMultiHoldingRegisters(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcWriteMultiHoldingRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcWriteMultiHoldingRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_funcReadWriteMultiHoldingRegisters(t *testing.T) {
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"数据长度data小于FuncReadWriteMinSize[9]",
			args{newNodeReg(),
				[]byte{0x00, 0x00, 0x00}},
			nil,
			true,
		},
		{
			"读数量小于1或大于125 或者 写数量小于1或大于121",
			args{newNodeReg(),
				[]byte{0x00, 0x00, 0x00, 0x7e, 0x00, 0x00, 0x00, 0x7a, 0x00}},
			nil,
			true,
		},
		{
			"读写值",
			args{newNodeReg(),
				[]byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x02, 0x12, 0x34}},
			[]byte{0x02, 0x12, 0x34},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcReadWriteMultiHoldingRegisters(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcReadWriteMultiHoldingRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcReadWriteMultiHoldingRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_funcMaskWriteRegisters(t *testing.T) {
	nodeReg := &NodeRegister{
		holdingAddrStart: 0,
		holding:          []uint16{0x0000, 0x0012, 0x0000},
	}
	type args struct {
		reg  *NodeRegister
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"数据长度data小于FuncMaskWriteMinSize[6]", args{newNodeReg(), []byte{0x00, 0x00, 0x00}}, nil, true},
		{"写值", args{nodeReg, []byte{0x00, 0x01, 0x00, 0xf2, 0x00, 0x25}}, []byte{0x00, 0x01, 0x00, 0xf2, 0x00, 0x25}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := funcMaskWriteRegisters(tt.args.reg, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("funcMaskWriteRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("funcMaskWriteRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}
