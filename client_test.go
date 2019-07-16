package modbus

import (
	"errors"
	"reflect"
	"testing"
)

// check implements ClientProvider interface
var _ ClientProvider = (*provider)(nil)

type provider struct {
	data []byte
	err  error
}

func (*provider) Connect() error             { return nil }
func (*provider) IsConnected() bool          { return true }
func (*provider) SetAutoReconnect(byte)      {}
func (*provider) LogMode(bool)               {}
func (*provider) SetLogProvider(LogProvider) {}
func (*provider) Close() error               { return nil }
func (r *provider) Send(_ byte, _ ProtocolDataUnit) (ProtocolDataUnit, error) {
	return ProtocolDataUnit{Data: r.data}, r.err
}
func (*provider) SendPdu(byte, []byte) (pduResponse []byte, err error) {
	return nil, nil
}
func (*provider) SendRawFrame([]byte) (aduResponse []byte, err error) {
	return nil, nil
}

func Test_client_ReadCoils(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-2000", &provider{},
			args{slaveID: 1, quantity: 20001}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x02, 0x00, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x02, 0x12, 0x34}},
			args{slaveID: 1, quantity: 10}, []byte{0x12, 0x34}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := NewClient(tt.provide)
			got, err := this.ReadCoils(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadCoils() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadCoils() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadDiscreteInputs(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-2000", &provider{},
			args{slaveID: 1, quantity: 20001}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x02, 0x12, 0x34}},
			args{slaveID: 1, quantity: 10}, []byte{0x12, 0x34}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadDiscreteInputs(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadDiscreteInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadDiscreteInputs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadHoldingRegistersBytes(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-125", &provider{},
			args{slaveID: 1, quantity: 126}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x04, 0x12, 0x34, 0x56, 0x78}},
			args{slaveID: 1, quantity: 2}, []byte{0x12, 0x34, 0x56, 0x78}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadHoldingRegistersBytes(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadHoldingRegistersBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadHoldingRegistersBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadHoldingRegisters(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []uint16
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-125", &provider{},
			args{slaveID: 1, quantity: 126}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x04, 0x12, 0x34, 0x56, 0x78}},
			args{slaveID: 1, quantity: 2}, []uint16{0x1234, 0x5678}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadHoldingRegisters(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadHoldingRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadHoldingRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadInputRegistersBytes(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-125", &provider{},
			args{slaveID: 1, quantity: 126}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x04, 0x12, 0x34, 0x56, 0x78}},
			args{slaveID: 1, quantity: 2}, []byte{0x12, 0x34, 0x56, 0x78}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadInputRegistersBytes(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadInputRegistersBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadInputRegistersBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadInputRegisters(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []uint16
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"Quantity不在范围1-125", &provider{},
			args{slaveID: 1, quantity: 126}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"返回字节与请求数量不符", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1, quantity: 10}, nil, true},
		{"正确", &provider{data: []byte{0x04, 0x12, 0x34, 0x56, 0x78}},
			args{slaveID: 1, quantity: 2}, []uint16{0x1234, 0x5678}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadInputRegisters(tt.args.slaveID, tt.args.address, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadInputRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadInputRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_WriteSingleCoil(t *testing.T) {
	type args struct {
		slaveID byte
		address uint16
		isOn    bool
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		wantErr bool
	}{
		{"slaveid不在范围0-247", &provider{},
			args{slaveID: 248}, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1}, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1}, true},
		{"返回字节不符合", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1}, true},
		{"返回地址不符合", &provider{data: []byte{0x00, 0x01, 0xff, 0x00}},
			args{slaveID: 1}, true},
		{"返回值不符合", &provider{data: []byte{0x00, 0x00, 0xff, 0x00}},
			args{slaveID: 1}, true},
		{"正确", &provider{data: []byte{0x00, 0x00, 0xff, 0x00}},
			args{slaveID: 1, isOn: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			if err := this.WriteSingleCoil(tt.args.slaveID, tt.args.address, tt.args.isOn); (err != nil) != tt.wantErr {
				t.Errorf("client.WriteSingleCoil() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_WriteSingleRegister(t *testing.T) {
	type args struct {
		slaveID byte
		address uint16
		value   uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		wantErr bool
	}{
		{"slaveid不在范围0-247", &provider{},
			args{slaveID: 248}, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1}, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1}, true},
		{"返回字节不符合", &provider{data: []byte{0x01, 0x00}},
			args{slaveID: 1}, true},
		{"返回地址不符合", &provider{data: []byte{0x00, 0x01, 0xff, 0x00}},
			args{slaveID: 1}, true},
		{"返回值不符合", &provider{data: []byte{0x00, 0x00, 0xff, 0x00}},
			args{slaveID: 1}, true},
		{"正确", &provider{data: []byte{0x00, 0x00, 0xff, 0x00}},
			args{slaveID: 1, value: 0xff00}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			if err := this.WriteSingleRegister(tt.args.slaveID, tt.args.address, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("client.WriteSingleRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_WriteMultipleCoils(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
		value    []byte
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		wantErr bool
	}{
		{"slaveid不在范围0-247", &provider{},
			args{slaveID: 248}, true},
		{"quantity不在范围1-1968", &provider{},
			args{quantity: 1969}, true},
		{"返回error", &provider{err: errors.New("error")},
			args{quantity: 1}, true},
		{"返回数据长度不符", &provider{data: []byte{0x00, 0x00, 0x00}},
			args{quantity: 1}, true},
		{"返回地址与请求一致", &provider{data: []byte{0x00, 0x01, 0x00, 0x01}},
			args{quantity: 1}, true},
		{"返回数量与请求不一致", &provider{data: []byte{0x00, 0x00, 0x00, 0x02}},
			args{quantity: 1}, true},
		{"正确", &provider{data: []byte{0x00, 0x00, 0x00, 0x01}},
			args{quantity: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			if err := this.WriteMultipleCoils(tt.args.slaveID, tt.args.address, tt.args.quantity, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("client.WriteMultipleCoils() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_WriteMultipleRegisters(t *testing.T) {
	type args struct {
		slaveID  byte
		address  uint16
		quantity uint16
		value    []byte
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		wantErr bool
	}{
		{"slaveid不在范围0-247", &provider{},
			args{slaveID: 248}, true},
		{"quantity不在范围1-123", &provider{},
			args{quantity: 124}, true},
		{"返回error", &provider{err: errors.New("error")},
			args{quantity: 1}, true},
		{"返回数据长度不符", &provider{data: []byte{0x00, 0x00, 0x00}},
			args{quantity: 1}, true},
		{"返回地址与请求一致", &provider{data: []byte{0x00, 0x01, 0x00, 0x01}},
			args{quantity: 1}, true},
		{"返回数量与请求不一致", &provider{data: []byte{0x00, 0x00, 0x00, 0x02}},
			args{quantity: 1}, true},
		{"正确", &provider{data: []byte{0x00, 0x00, 0x00, 0x01}},
			args{quantity: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			if err := this.WriteMultipleRegisters(tt.args.slaveID, tt.args.address, tt.args.quantity, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("client.WriteMultipleRegisters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_MaskWriteRegister(t *testing.T) {
	type args struct {
		slaveID byte
		address uint16
		andMask uint16
		orMask  uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		wantErr bool
	}{
		{"slaveid不在范围0-247", &provider{},
			args{slaveID: 248}, true},
		{"返回error", &provider{err: errors.New("error")},
			args{}, true},
		{"返回数据长度不符", &provider{data: []byte{0x00, 0x00, 0x00}},
			args{}, true},
		{"返回地址与请求一致", &provider{data: []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03}},
			args{address: 0, andMask: 0x0002, orMask: 0x0003}, true},
		{"返回andMask与请求不一致", &provider{data: []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03}},
			args{address: 1, andMask: 0x0003, orMask: 0x0003}, true},
		{"返回orMask与请求不一致", &provider{data: []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03}},
			args{address: 1, andMask: 0x0002, orMask: 0x0004}, true},
		{"正确", &provider{data: []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03}},
			args{address: 1, andMask: 0x0002, orMask: 0x0003}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			if err := this.MaskWriteRegister(tt.args.slaveID, tt.args.address, tt.args.andMask, tt.args.orMask); (err != nil) != tt.wantErr {
				t.Errorf("client.MaskWriteRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_ReadWriteMultipleRegistersBytes(t *testing.T) {
	type args struct {
		slaveID       byte
		readAddress   uint16
		readQuantity  uint16
		writeAddress  uint16
		writeQuantity uint16
		value         []byte
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"读数量不在范围1-125", &provider{},
			args{slaveID: 1, readQuantity: 126}, nil, true},
		{"读数量不在范围1-123", &provider{},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 124}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 1}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 1}, nil, true},
		{"正确", &provider{data: []byte{0x02, 0x00, 0x03}},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 1}, []byte{0x00, 0x03}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadWriteMultipleRegistersBytes(tt.args.slaveID, tt.args.readAddress, tt.args.readQuantity, tt.args.writeAddress, tt.args.writeQuantity, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadWriteMultipleRegistersBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadWriteMultipleRegistersBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadWriteMultipleRegisters(t *testing.T) {
	type args struct {
		slaveID       byte
		readAddress   uint16
		readQuantity  uint16
		writeAddress  uint16
		writeQuantity uint16
		value         []byte
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []uint16
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"读数量不在范围1-125", &provider{},
			args{slaveID: 1, readQuantity: 126}, nil, true},
		{"写数量不在范围1-123", &provider{},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 124}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 124}, nil, true},
		{"返回数据长度不符", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 1}, nil, true},
		{"正确", &provider{data: []byte{0x02, 0x00, 0x03}},
			args{slaveID: 1, readQuantity: 1, writeQuantity: 1}, []uint16{0x0003}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadWriteMultipleRegisters(tt.args.slaveID, tt.args.readAddress, tt.args.readQuantity, tt.args.writeAddress, tt.args.writeQuantity, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadWriteMultipleRegisters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadWriteMultipleRegisters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_ReadFIFOQueue(t *testing.T) {
	type args struct {
		slaveID byte
		address uint16
	}
	tests := []struct {
		name    string
		provide ClientProvider
		args    args
		want    []byte
		wantErr bool
	}{
		{"slaveid不在范围1-247", &provider{},
			args{slaveID: 248}, nil, true},
		{"返回error", &provider{err: errors.New("error")},
			args{slaveID: 1}, nil, true},
		{"返回数据长度不符,需大于4", &provider{data: []byte{0x01, 0x00, 0x00}},
			args{slaveID: 1}, nil, true},
		{"byte长度不正确", &provider{data: []byte{0x00, 0x02, 0x00, 0x01, 0x02}},
			args{slaveID: 1}, nil, true},
		{"fifo长度超范围0-31", &provider{data: []byte{0x00, 0x02, 0x00, 0x20}},
			args{slaveID: 1}, nil, true},
		{"正确", &provider{data: []byte{0x00, 0x04, 0x00, 0x01, 0x01, 0x02}},
			args{slaveID: 1}, []byte{0x01, 0x02}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := &client{
				ClientProvider: tt.provide,
			}
			got, err := this.ReadFIFOQueue(tt.args.slaveID, tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ReadFIFOQueue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ReadFIFOQueue() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
