package modbus

import (
	"reflect"
	"testing"
)

func TestASCIIClientProvider_encode(t *testing.T) {
	type args struct {
		slaveID byte
		pdu     *ProtocolDataUnit
	}
	tests := []struct {
		name    string
		ascii   *protocolASCIIFrame
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"ASCII encode 1",
			&protocolASCIIFrame{},
			args{8, &ProtocolDataUnit{1, []byte{2, 66, 1, 5}}},
			[]byte(":080102420105AD\r\n"),
			false,
		},
		{
			"ASCII encode 2",
			&protocolASCIIFrame{},
			args{1, &ProtocolDataUnit{3, []byte{8, 100, 10, 13}}},
			[]byte(":010308640A0D79\r\n"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ascii.encode(tt.args.slaveID, tt.args.pdu)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASCIIClientProvider.encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ASCIIClientProvider.encode() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestASCIIClientProvider_decode(t *testing.T) {
	type args struct {
		adu []byte
	}
	tests := []struct {
		name    string
		ascii   *protocolASCIIFrame
		args    args
		slaveID uint8
		pdu     *ProtocolDataUnit
		wantErr bool
	}{
		{
			"ASCII decode 1",
			&protocolASCIIFrame{},
			args{[]byte(":080102420105AD\r\n")},
			8,
			&ProtocolDataUnit{1, []byte{2, 66, 1, 5}},
			false,
		},
		{
			"ASCII decode 2",
			&protocolASCIIFrame{},
			args{[]byte(":010308640A0D79\r\n")},
			1,
			&ProtocolDataUnit{3, []byte{8, 100, 10, 13}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotslaveID, gotpdu, _, err := tt.ascii.decode(tt.args.adu)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASCIIClientProvider.decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotslaveID != tt.slaveID {
				t.Errorf("ASCIIClientProvider.decode() gotslaveID = %v, want %v", gotslaveID, tt.slaveID)
			}
			if !reflect.DeepEqual(gotpdu, tt.pdu) {
				t.Errorf("ASCIIClientProvider.decode() gotpdu = %v, want %v", gotpdu, tt.pdu)
			}
		})
	}
}

func BenchmarkASCIIClientProvider_encode(b *testing.B) {
	p := protocolASCIIFrame{}
	pdu := &ProtocolDataUnit{
		FuncCode: 1,
		Data:     []byte{2, 3, 4, 5, 6, 7, 8, 9},
	}
	for i := 0; i < b.N; i++ {
		_, err := p.encode(10, pdu)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkASCIIClientProvider_decode(b *testing.B) {
	p := protocolASCIIFrame{}
	adu := []byte(":010308640A0D79\r\n")
	for i := 0; i < b.N; i++ {
		_, _, _, err := p.decode(adu)
		if err != nil {
			b.Fatal(err)
		}
	}
}
