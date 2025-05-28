// serialcomm/serialcomm.go
package serialcomm

import (
	"time"
)

type MessageHandler func(msg *Message, payload *Payload)

type SerialConfig struct {
	PortName     string
	BaudRate     int
	ReadTimeout  time.Duration
	MaxLength    int
	ReadCallback MessageHandler
}

type SerialReceiver interface {
	Start() error
	Close() error
}

type SerialSender interface {
	Send(data []byte) error
	Close() error
}
