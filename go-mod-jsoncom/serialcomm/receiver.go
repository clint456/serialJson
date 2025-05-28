// serialcomm/receiver.go
package serialcomm

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log"
	"time"

	"github.com/tarm/serial"
)

type serialReceiverImpl struct {
	port    *serial.Port
	config  *SerialConfig
	stopCh  chan struct{}
	started bool
}

func NewSerialReceiver(cfg *SerialConfig) (SerialReceiver, error) {
	portCfg := &serial.Config{
		Name:        cfg.PortName,
		Baud:        cfg.BaudRate,
		Parity:      serial.ParityNone,
		ReadTimeout: cfg.ReadTimeout,
	}
	port, err := serial.OpenPort(portCfg)
	if err != nil {
		return nil, err
	}
	return &serialReceiverImpl{
		port:   port,
		config: cfg,
		stopCh: make(chan struct{}),
	}, nil
}

func (s *serialReceiverImpl) Start() error {
	var (
		buffer         bytes.Buffer
		data           = make([]byte, 1024)
		expectedLength uint32
		lastDataTime   = time.Now()
		timeout        = 5 * time.Second
	)

	go func() {
		defer s.port.Close()
		log.Println("串口监听启动")

		for {
			select {
			case <-s.stopCh:
				log.Println("串口监听停止")
				return
			default:
			}

			n, err := s.port.Read(data)
			if err != nil {
				log.Printf("读取错误: %v", err)
				continue
			}
			if n == 0 {
				if time.Since(lastDataTime) > timeout && buffer.Len() > 0 {
					log.Println("接收超时，重置状态")
					buffer.Reset()
					expectedLength = 0
					_ = sendFeedback(s.port, "RETRY")
				}
				continue
			}

			lastDataTime = time.Now()
			buffer.Write(data[:n])

			// 尝试读取长度和 CRC 校验
			if expectedLength == 0 && buffer.Len() >= 4 {
				expectedLength = binary.BigEndian.Uint32(buffer.Next(4))
				if expectedLength > uint32(s.config.MaxLength) || expectedLength == 0 {
					buffer.Reset()
					expectedLength = 0
					_ = sendFeedback(s.port, "RETRY")
					continue
				}
			}

			if expectedLength > 0 && buffer.Len() >= int(expectedLength)+2 {
				dataPacket := buffer.Next(int(expectedLength))
				crcBytes := buffer.Next(2)
				receivedCRC := binary.BigEndian.Uint16(crcBytes)
				calculatedCRC := calculateCRC16(dataPacket)

				if receivedCRC != calculatedCRC {
					log.Println("CRC 校验失败")
					buffer.Reset()
					expectedLength = 0
					_ = sendFeedback(s.port, "RETRY")
					continue
				}

				var msg Message
				if err := json.Unmarshal(dataPacket, &msg); err != nil {
					log.Printf("消息解码失败: %v", err)
					buffer.Reset()
					expectedLength = 0
					_ = sendFeedback(s.port, "RETRY")
					continue
				}

				// Base64解码 payload
				payloadBytes, err := base64.StdEncoding.DecodeString(msg.Payload)
				if err != nil {
					log.Printf("Payload base64 解码失败: %v", err)
					continue
				}
				var payload Payload
				if err := json.Unmarshal(payloadBytes, &payload); err != nil {
					log.Printf("Payload JSON 解码失败: %v", err)
					continue
				}

				// 触发回调
				if s.config.ReadCallback != nil {
					s.config.ReadCallback(&msg, &payload)
				}

				_ = sendFeedback(s.port, "OK")
				buffer.Reset()
				expectedLength = 0
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	s.started = true
	return nil
}

func (s *serialReceiverImpl) Close() error {
	if s.started {
		close(s.stopCh)
	}
	return nil
}
