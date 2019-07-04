package modbus

import (
	"reflect"
	"testing"
)

const (
	testslaveID1 = 0x01
	testslaveID2 = 0x02
)

func Test_TCPClientWithServer(t *testing.T) {
	t.Run("", func(t *testing.T) {
		mbsrv := NewTCPServer(":505")
		mbsrv.AddNodes(NewNodeRegister(testslaveID1, 0, 10, 0, 10,
			0, 10, 0, 10))
		mbsrv.AddNodes(NewNodeRegister(testslaveID2, 0, 10, 0, 10,
			0, 10, 0, 10))

		_, err := mbsrv.GetNode(testslaveID2)
		if err != nil {
			t.Errorf("GetNode(%#v) error = %v, wantErr %v", testslaveID2, err, nil)
			return
		}

		list := mbsrv.GetNodeList()
		if list == nil {
			t.Errorf("GetNodeList() should not nil")
			return
		}

		mbsrv.DeleteNode(testslaveID2)
		_, err = mbsrv.GetNode(testslaveID2)
		if err == nil {
			t.Errorf("GetNode(%#v) error = %v, wantErr %v", testslaveID2, err, "slaveID not exist")
			return
		}

		go mbsrv.ServerModbus()

		mbPro := NewTCPClientProvider("localhost:505")
		mbCli := NewClient(mbPro)
		err = mbCli.Connect()
		if err != nil {
			t.Errorf("Connect error = %v, wantErr %v", err, nil)
			return
		}

		result, err := mbCli.ReadCoils(testslaveID1, 0, 10)
		if err != nil {
			t.Errorf("ReadCoils error = %v, wantErr %v", err, nil)
			return
		}

		if !reflect.DeepEqual(result, []byte{0x00, 0x00}) {
			t.Errorf("ReadCoils result = %#v, want %#v", result, []byte{0x00, 0x00})
		}

		if !mbCli.IsConnected() {
			t.Errorf("client IsConnected() = %v, want %v", false, true)
			return
		}

		err = mbCli.Close()
		if err != nil {
			t.Errorf("client Close() error = %v, wantErr %v", err, nil)
			return
		}

		if mbCli.IsConnected() {
			t.Errorf("client IsConnected() = %v, want %v", true, false)
			return
		}

		err = mbsrv.Close()
		if err != nil {
			t.Errorf("server Close() error = %v, wantErr %v", err, nil)
			return
		}
	})
}
