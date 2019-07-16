package modbus

import (
	"reflect"
	"testing"
)

func Test_protocolFrame_encodeTCPFrame(t *testing.T) {
	newBuffer := func() *protocolFrame {
		return &protocolFrame{make([]byte, 0, tcpAduMaxSize)}
	}

	type args struct {
		tid     uint16
		slaveID byte
		pdu     ProtocolDataUnit
	}
	tests := []struct {
		name    string
		this    *protocolFrame
		args    args
		want    protocolTCPHeader
		want1   []byte
		wantErr bool
	}{
		{
			"TCP encode",
			newBuffer(),
			args{
				0,
				0,
				ProtocolDataUnit{1, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}}},
			protocolTCPHeader{0, 0, 11, 0},
			[]byte{0, 0, 0, 0, 0, 11, 0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.this.encodeTCPFrame(tt.args.tid, tt.args.slaveID, tt.args.pdu)
			if (err != nil) != tt.wantErr {
				t.Errorf("protocolFrame.encodeTCPFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("protocolFrame.encodeTCPFrame() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("protocolFrame.encodeTCPFrame() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTCPClientProvider_decodeTCPFrame(t *testing.T) {
	type args struct {
		adu []byte
	}
	tests := []struct {
		name    string
		args    args
		head    protocolTCPHeader
		pdu     []byte
		wantErr bool
	}{
		{
			"TCP decode",
			args{[]byte{0, 0, 0, 0, 0, 11, 0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			protocolTCPHeader{0, 0, 11, 0},
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

func Test_verifyTCPFrame(t *testing.T) {
	type args struct {
		reqHead protocolTCPHeader
		rspHead protocolTCPHeader
		reqPDU  ProtocolDataUnit
		rspPDU  ProtocolDataUnit
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"TCP verify same",
			args{
				protocolTCPHeader{1, 2, 6, 4},
				protocolTCPHeader{1, 2, 6, 4},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			false,
		},
		{
			"TCP verify transactionID different",
			args{
				protocolTCPHeader{1, 2, 6, 4},
				protocolTCPHeader{5, 2, 6, 4},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"TCP verify protocolID different",
			args{
				protocolTCPHeader{1, 2, 6, 4},
				protocolTCPHeader{1, 5, 6, 4},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify slaveID different",
			args{
				protocolTCPHeader{1, 2, 6, 11},
				protocolTCPHeader{1, 2, 6, 4},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify functionCode different",
			args{
				protocolTCPHeader{1, 2, 6, 4},
				protocolTCPHeader{1, 2, 6, 4},
				ProtocolDataUnit{10, []byte{1, 2, 3, 4}},
				ProtocolDataUnit{11, []byte{1, 2, 3, 4}},
			},
			true,
		},
		{
			"serial verify pdu data zero length",
			args{
				protocolTCPHeader{1, 2, 6, 4},
				protocolTCPHeader{1, 2, 6, 4},
				ProtocolDataUnit{10, []byte{}},
				ProtocolDataUnit{10, []byte{}},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifyTCPFrame(tt.args.reqHead, tt.args.rspHead, tt.args.reqPDU, tt.args.rspPDU); (err != nil) != tt.wantErr {
				t.Errorf("verifyTCPFrame() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkTCPClientProvider_encodeTCPFrame(b *testing.B) {
	tcp := &protocolFrame{make([]byte, 0, tcpAduMaxSize)}
	pdu := ProtocolDataUnit{
		1,
		[]byte{2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	for i := 0; i < b.N; i++ {
		_, _, err := tcp.encodeTCPFrame(0, 0, pdu)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTCPClientProvider_decodeTCPFrame(b *testing.B) {
	adu := []byte{0, 1, 0, 0, 0, 9, 20, 1, 2, 3, 4, 5, 6, 7, 8}
	for i := 0; i < b.N; i++ {
		_, _, err := decodeTCPFrame(adu)
		if err != nil {
			b.Fatal(err)
		}
	}
}
