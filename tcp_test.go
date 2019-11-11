package modbus

import (
	"reflect"
	"testing"
	"time"
)

const (
	testslaveID1 = 0x01
	testslaveID2 = 0x02
)

func Test_TCPClientWithServer(t *testing.T) {
	t.Run("", func(t *testing.T) {
		mbSrv := NewTCPServer()
		mbSrv.AddNodes(NewNodeRegister(testslaveID1,
			0, 10, 0, 10,
			0, 10, 0, 10),
			NewNodeRegister(testslaveID2,
				0, 10, 0, 10,
				0, 10, 0, 10))

		_, err := mbSrv.GetNode(testslaveID2)
		if err != nil {
			t.Errorf("GetNode(%#v) error = %v, wantErr %v", testslaveID2, err, nil)
			return
		}

		list := mbSrv.GetNodeList()
		if list == nil {
			t.Errorf("GetNodeList() should not nil")
			return
		}

		mbSrv.DeleteNode(testslaveID2)
		_, err = mbSrv.GetNode(testslaveID2)
		if err == nil {
			t.Errorf("GetNode(%#v) error = %v, wantErr %v", testslaveID2, err, "slaveID not exist")
			return
		}

		go mbSrv.ListenAndServe("localhost:48091")
		time.Sleep(time.Second) // 让服务器完全启动
		mbPro := NewTCPClientProvider("localhost:48091")
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

		err = mbSrv.Close()
		if err != nil {
			t.Errorf("server Close() error = %v, wantErr %v", err, nil)
			return
		}
	})
}
