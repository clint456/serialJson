// serialcomm/utils.go
package serialcomm

import (
	"github.com/sigurn/crc16"
	"github.com/tarm/serial"
)

func calculateCRC16(data []byte) uint16 {
	table := crc16.MakeTable(crc16.CRC16_MODBUS)
	return crc16.Checksum(data, table)
}

func sendFeedback(port *serial.Port, msg string) error {
	_, err := port.Write([]byte(msg))
	return err
}
