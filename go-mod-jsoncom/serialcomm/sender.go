// serialcomm/sender.go
package serialcomm

import (
	"bytes"
	"encoding/binary"

	"github.com/tarm/serial"
)

type serialSenderImpl struct {
	port *serial.Port
}

func NewSerialSender(portName string, baud int) (SerialSender, error) {
	cfg := &serial.Config{Name: portName, Baud: baud}
	port, err := serial.OpenPort(cfg)
	if err != nil {
		return nil, err
	}
	return &serialSenderImpl{port: port}, nil
}

func (s *serialSenderImpl) Send(data []byte) error {
	crc := calculateCRC16(data)
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)
	_ = binary.Write(buf, binary.BigEndian, crc)

	_, err := s.port.Write(buf.Bytes())
	return err
}

func (s *serialSenderImpl) Close() error {
	return s.port.Close()
}
