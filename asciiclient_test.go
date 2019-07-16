package modbus

import (
	"reflect"
	"testing"
)

func TestASCIIClientProvider_encodeASCIIFrame(t *testing.T) {
	type args struct {
		slaveID byte
		pdu     ProtocolDataUnit
	}
	tests := []struct {
		name    string
		ascii   *protocolFrame
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"ASCII encode right 1",
			&protocolFrame{adu: make([]byte, 0, asciiCharacterMaxSize)},
			args{8, ProtocolDataUnit{1, []byte{2, 66, 1, 5}}},
			[]byte(":080102420105AD\r\n"),
			false,
		},
		{
			"ASCII encode right 2",
			&protocolFrame{adu: make([]byte, 0, asciiCharacterMaxSize)},
			args{1, ProtocolDataUnit{3, []byte{8, 100, 10, 13}}},
			[]byte(":010308640A0D79\r\n"),
			false,
		},
		{
			"ASCII encode error",
			&protocolFrame{adu: make([]byte, 0, asciiCharacterMaxSize)},
			args{1, ProtocolDataUnit{3, make([]byte, 254)}},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ascii.encodeASCIIFrame(tt.args.slaveID, tt.args.pdu)
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

func TestASCIIClientProvider_decodeASCIIFrame(t *testing.T) {
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
			"ASCII decode 1",
			args{[]byte(":080102420105AD\r\n")},
			8,
			[]byte{1, 2, 66, 1, 5},
			false,
		},
		{
			"ASCII decode 2",
			args{[]byte(":010308640A0D79\r\n")},
			1,
			[]byte{3, 8, 100, 10, 13},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotslaveID, gotpdu, err := decodeASCIIFrame(tt.args.adu)
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

func BenchmarkASCIIClientProvider_encodeASCIIFrame(b *testing.B) {
	p := protocolFrame{adu: make([]byte, 0, asciiCharacterMaxSize)}
	pdu := ProtocolDataUnit{
		1,
		[]byte{2, 3, 4, 5, 6, 7, 8, 9},
	}
	for i := 0; i < b.N; i++ {
		_, err := p.encodeASCIIFrame(10, pdu)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkASCIIClientProvider_decodeASCIIFrame(b *testing.B) {
	adu := []byte(":010308640A0D79\r\n")
	for i := 0; i < b.N; i++ {
		_, _, err := decodeASCIIFrame(adu)
		if err != nil {
			b.Fatal(err)
		}
	}
}
