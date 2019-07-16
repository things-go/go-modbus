package modbus

import (
	"reflect"
	"testing"
)

func TestRTUClientProvider_encodeRTUFrame(t *testing.T) {
	type args struct {
		slaveID byte
		pdu     ProtocolDataUnit
	}
	tests := []struct {
		name    string
		rtu     *protocolFrame
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"RTU encode",
			&protocolFrame{make([]byte, 0, rtuAduMaxSize)},
			args{0x01, ProtocolDataUnit{0x03, []byte{0x01, 0x02, 0x03, 0x04, 0x05}}},
			[]byte{0x01, 0x03, 0x01, 0x02, 0x03, 0x04, 0x05, 0x05, 0x48},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rtu.encodeRTUFrame(tt.args.slaveID, tt.args.pdu)
			if (err != nil) != tt.wantErr {
				t.Errorf("RTUClientProvider.encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RTUClientProvider.encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRTUClientProvider_decodeRTUFrame(t *testing.T) {
	type args struct {
		adu []byte
	}
	tests := []struct {
		name    string
		args    args
		slaveID uint8
		pdu     []byte
		wantErr bool
	}{
		{
			"RTU decode",
			args{[]byte{0x01, 0x03, 0x01, 0x02, 0x03, 0x04, 0x05, 0x05, 0x48}},
			0x01,
			[]byte{0x03, 0x01, 0x02, 0x03, 0x04, 0x05},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotslaveID, gotpdu, err := decodeRTUFrame(tt.args.adu)
			if (err != nil) != tt.wantErr {
				t.Errorf("RTUClientProvider.decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotslaveID != tt.slaveID {
				t.Errorf("RTUClientProvider.decode() gotslaveID = %v, want %v", gotslaveID, tt.slaveID)
			}
			if !reflect.DeepEqual(gotpdu, tt.pdu) {
				t.Errorf("RTUClientProvider.decode() gotpdu = %v, want %v", gotpdu, tt.pdu)
			}
		})
	}
}

func Test_verify(t *testing.T) {
	type args struct {
		reqSlaveID uint8
		rspSlaveID uint8
		reqPDU     ProtocolDataUnit
		rspPDU     ProtocolDataUnit
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"serial verify same",
			args{
				5,
				5,
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			false,
		},
		{
			"serial verify slaveID different",
			args{
				4,
				5,
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify functionCode different",
			args{
				5,
				5,
				ProtocolDataUnit{11, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify pdu data zero length",
			args{
				5,
				5,
				ProtocolDataUnit{10, []byte{}},
				ProtocolDataUnit{10, []byte{}},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verify(tt.args.reqSlaveID, tt.args.rspSlaveID, tt.args.reqPDU, tt.args.rspPDU); (err != nil) != tt.wantErr {
				t.Errorf("verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_calculateResponseLength(t *testing.T) {
	type args struct {
		adu []byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"1", args{[]byte{4, 1, 0, 0xA, 0, 0xD, 0xDD, 0x98}}, 7},
		{"2", args{[]byte{4, 2, 0, 0xA, 0, 0xD, 0x99, 0x98}}, 7},
		{"3", args{[]byte{1, 3, 0, 0, 0, 2, 0xC4, 0xB}}, 9},
		{"4", args{[]byte{0x11, 5, 0, 0xAC, 0xFF, 0, 0x4E, 0x8B}}, 8},
		{"5", args{[]byte{0x11, 6, 0, 1, 0, 3, 0x9A, 0x9B}}, 8},
		{"6", args{[]byte{0x11, 0xF, 0, 0x13, 0, 0xA, 2, 0xCD, 1, 0xBF, 0xB}}, 8},
		{"7", args{[]byte{0x11, 0x10, 0, 1, 0, 2, 4, 0, 0xA, 1, 2, 0xC6, 0xF0}}, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateResponseLength(tt.args.adu); got != tt.want {
				t.Errorf("calculateResponseLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkRTUClientProvider_encodeRTUFrame(b *testing.B) {
	p := &protocolFrame{make([]byte, 0, rtuAduMaxSize)}
	pdu := ProtocolDataUnit{
		1,
		[]byte{2, 3, 4, 5, 6, 7, 8, 9},
	}
	for i := 0; i < b.N; i++ {
		_, err := p.encodeRTUFrame(10, pdu)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRTUClientProvider_decodeRTUFrame(b *testing.B) {
	adu := []byte{0x01, 0x10, 0x8A, 0x00, 0x00, 0x03, 0xAA, 0x10}
	for i := 0; i < b.N; i++ {
		_, _, err := decodeRTUFrame(adu)
		if err != nil {
			b.Fatal(err)
		}
	}
}
