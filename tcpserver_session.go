package modbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ServerSession tcp server session
type ServerSession struct {
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	*serverCommon
	logger
}

// handler net conn
func (sf *ServerSession) running(ctx context.Context) {
	var err error
	var bytesRead int

	sf.Debugf("client(%v) -> server(%v) connected", sf.conn.RemoteAddr(), sf.conn.LocalAddr())
	defer func() {
		sf.conn.Close()
		sf.Debugf("client(%v) -> server(%v) disconnected,cause by %v", sf.conn.RemoteAddr(), sf.conn.LocalAddr(), err)
	}()

	raw := make([]byte, tcpAduMaxSize)
	for {
		select {
		case <-ctx.Done():
			err = errors.New("server active close")
			return
		default:
		}

		adu := raw
		for rdCnt, length := 0, tcpHeaderMbapSize; rdCnt < length; {
			err = sf.conn.SetReadDeadline(time.Now().Add(sf.readTimeout))
			if err != nil {
				return
			}
			bytesRead, err = io.ReadFull(sf.conn, adu[rdCnt:length])
			if err != nil {
				if err != io.EOF && err != io.ErrClosedPipe || strings.Contains(err.Error(), "use of closed network connection") {
					return
				}

				if e, ok := err.(net.Error); ok && !e.Temporary() {
					return
				}

				if bytesRead == 0 && err == io.EOF {
					err = fmt.Errorf("remote client closed, %v", err)
					return
				}
				// cnt >0 do nothing
				// cnt == 0 && err != io.EOF continue do it next
			}
			rdCnt += bytesRead
			if rdCnt >= length {
				// check head ProtocolIdentifier
				if binary.BigEndian.Uint16(adu[2:]) != tcpProtocolIdentifier {
					rdCnt, length = 0, tcpHeaderMbapSize
					continue
				}
				length = int(binary.BigEndian.Uint16(adu[4:])) + tcpHeaderMbapSize - 1
				if rdCnt == length {
					if err = sf.frameHandler(adu[:length]); err != nil {
						return
					}
				}
			}
		}
	}
}

// modbus 包处理
func (sf *ServerSession) frameHandler(requestAdu []byte) error {
	defer func() {
		if err := recover(); err != nil {
			sf.Errorf("painc happen,%v", err)
		}
	}()

	sf.Debugf("RX Raw[% x]", requestAdu)
	// got head from request adu
	tcpHeader := protocolTCPHeader{
		binary.BigEndian.Uint16(requestAdu[0:]),
		binary.BigEndian.Uint16(requestAdu[2:]),
		binary.BigEndian.Uint16(requestAdu[4:]),
		requestAdu[6],
	}
	funcCode := requestAdu[7]
	pduData := requestAdu[8:]

	node, err := sf.GetNode(tcpHeader.slaveID)
	if err != nil { // slave id not exit, ignore it
		return nil
	}
	var rspPduData []byte
	if handle, ok := sf.function[funcCode]; ok {
		rspPduData, err = handle(node, pduData)
	} else {
		err = &ExceptionError{ExceptionCodeIllegalFunction}
	}
	if err != nil {
		funcCode |= 0x80
		rspPduData = []byte{err.(*ExceptionError).ExceptionCode}
	}

	// prepare responseAdu data,fill it
	responseAdu := requestAdu[:tcpHeaderMbapSize]
	binary.BigEndian.PutUint16(responseAdu[0:], tcpHeader.transactionID)
	binary.BigEndian.PutUint16(responseAdu[2:], tcpHeader.protocolID)
	binary.BigEndian.PutUint16(responseAdu[4:], uint16(2+len(rspPduData)))
	responseAdu[6] = tcpHeader.slaveID
	responseAdu = append(responseAdu, funcCode)
	responseAdu = append(responseAdu, rspPduData...)

	sf.Debugf("TX Raw[% x]", responseAdu)
	// write response
	return func(b []byte) error {
		for wrCnt := 0; len(b) > wrCnt; {
			err = sf.conn.SetWriteDeadline(time.Now().Add(sf.writeTimeout))
			if err != nil {
				return fmt.Errorf("set read deadline %v", err)
			}
			byteCount, err := sf.conn.Write(b[wrCnt:])
			if err != nil {
				// See: https://github.com/golang/go/issues/4373
				if err != io.EOF && err != io.ErrClosedPipe ||
					strings.Contains(err.Error(), "use of closed network connection") {
					return err
				}
				if e, ok := err.(net.Error); !ok || !e.Temporary() {
					return err
				}
				// temporary error may be recoverable
			}
			wrCnt += byteCount
		}
		return nil
	}(responseAdu)
}
