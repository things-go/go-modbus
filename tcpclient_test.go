package modbus

import (
	"reflect"
	"testing"
)

func Test_protocolTCPFrame_encode(t *testing.T) {
	type args struct {
		slaveID byte
		pdu     *ProtocolDataUnit
	}
	tests := []struct {
		name    string
		this    *protocolTCPFrame
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"TCP encode",
			&protocolTCPFrame{},
			args{
				0,
				&ProtocolDataUnit{1, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
			},
			[]byte{0, 0, 0, 0, 0, 11, 0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.this.encode(tt.args.slaveID, tt.args.pdu)
			if (err != nil) != tt.wantErr {
				t.Errorf("protocolTCPFrame.encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("protocolTCPFrame.encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPClientProvider_decode(t *testing.T) {
	type args struct {
		adu []byte
	}
	tests := []struct {
		name    string
		args    args
		head    *protocolTCPHeader
		pdu     []byte
		wantErr bool
	}{
		{
			"TCP decode",
			args{[]byte{0, 0, 0, 0, 0, 11, 0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			&protocolTCPHeader{0, 0, 11, 0},
			[]byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gothead, gotpdu, err := decodeTCPFrame(tt.args.adu)
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPClientProvider.decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gothead, tt.head) {
				t.Errorf("TCPClientProvider.decode() gothead = %v, want %v", gothead, tt.head)
			}
			if !reflect.DeepEqual(gotpdu, tt.pdu) {
				t.Errorf("TCPClientProvider.decode() gotpdu = %v, want %v", gotpdu, tt.pdu)
			}
		})
	}
}

func Test_protocolTCPFrame_verify(t *testing.T) {
	type args struct {
		rspHead *protocolTCPHeader
		rspPDU  *ProtocolDataUnit
	}
	tests := []struct {
		name    string
		this    *protocolTCPFrame
		args    args
		wantErr bool
	}{
		{
			"TCP verify same",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 4},
				pdu:  ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			args{
				&protocolTCPHeader{1, 2, 6, 4},
				&ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			false,
		},
		{
			"TCP verify transactionID different",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 4},
				pdu:  ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			args{
				&protocolTCPHeader{5, 2, 6, 4},
				&ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"TCP verify protocolID different",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 4},
				pdu:  ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			args{
				&protocolTCPHeader{1, 5, 6, 4},
				&ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify slaveID different",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 11},
				pdu:  ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			args{
				&protocolTCPHeader{1, 2, 6, 4},
				&ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify functionCode different",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 4},
				pdu:  ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			args{
				&protocolTCPHeader{1, 2, 6, 4},
				&ProtocolDataUnit{11, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify pdu data zero length",
			&protocolTCPFrame{
				head: protocolTCPHeader{1, 2, 6, 4},
				pdu:  ProtocolDataUnit{10, []byte{}},
			},
			args{
				&protocolTCPHeader{1, 2, 6, 4},
				&ProtocolDataUnit{10, []byte{}},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.this.verify(tt.args.rspHead, tt.args.rspPDU); (err != nil) != tt.wantErr {
				t.Errorf("protocolTCPFrame.verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkTCPClientProvider_encoder(b *testing.B) {
	tcp := &protocolTCPFrame{}
	pdu := &ProtocolDataUnit{
		FuncCode: 1,
		Data:     []byte{2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	for i := 0; i < b.N; i++ {
		_, err := tcp.encode(0, pdu)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTCPClientProvider_decoder(b *testing.B) {
	adu := []byte{0, 1, 0, 0, 0, 9, 20, 1, 2, 3, 4, 5, 6, 7, 8}
	for i := 0; i < b.N; i++ {
		_, _, err := decodeTCPFrame(adu)
		if err != nil {
			b.Fatal(err)
		}
	}
}
