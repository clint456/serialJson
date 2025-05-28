package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sigurn/crc16"
	"github.com/tarm/serial"
)

type Reading struct {
	ID           string `json:"id"`
	Origin       int64  `json:"origin"`
	DeviceName   string `json:"deviceName"`
	ResourceName string `json:"resourceName"`
	ProfileName  string `json:"profileName"`
	ValueType    string `json:"valueType"`
	Value        string `json:"value"`
}

type Event struct {
	APIVersion  string    `json:"apiVersion"`
	ID          string    `json:"id"`
	DeviceName  string    `json:"deviceName"`
	ProfileName string    `json:"profileName"`
	SourceName  string    `json:"sourceName"`
	Origin      int64     `json:"origin"`
	Readings    []Reading `json:"readings"`
}

type Payload struct {
	APIVersion string `json:"apiVersion"`
	RequestID  string `json:"requestID"`
	Event      Event  `json:"event"`
}

type Message struct {
	APIVersion    string `json:"apiVersion"`
	ReceivedTopic string `json:"receivedTopic"`
	CorrelationID string `json:"correlationID"`
	RequestID     string `json:"requestID"`
	ErrorCode     int    `json:"errorCode"`
	Payload       string `json:"payload"`
	ContentType   string `json:"contentType"`
}

func calculateCRC16(data []byte) uint16 {
	table := crc16.MakeTable(crc16.CRC16_MODBUS)
	if table == nil {
		log.Fatal("无法生成CRC16表")
	}
	crc := crc16.Checksum(data, table)
	log.Printf("CRC16 计算输入数据长度: %d, 校验和: %x", len(data), crc)
	return crc
}

func sendData(port *serial.Port, data []byte) error {
	// 添加4字节长度前缀（大端序）
	length := uint32(len(data))
	log.Printf("长度前缀的值为:%v", length)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	log.Printf("长度前缀数组为:%v", lengthBytes)
	_, err := port.Write(lengthBytes)
	if err != nil {
		return fmt.Errorf("发送长度前缀失败: %v", err)
	}
	log.Printf("发送长度前缀: %d字节（十六进制: %x）", length, lengthBytes)

	// 计算CRC16校验和
	crc := calculateCRC16(data)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)

	// 按20字节分段发送
	chunkSize := 20
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		_, err = port.Write(chunk)
		if err != nil {
			return fmt.Errorf("发送第%d块数据失败: %v", i/chunkSize+1, err)
		}
		log.Printf("发送第%d块数据: %d字节，内容: %q (十六进制: %x)", i/chunkSize+1, len(chunk), chunk, chunk)
		time.Sleep(50 * time.Millisecond) // 每段之间添加50ms延迟
	}

	// 发送CRC16校验和
	_, err = port.Write(crcBytes)
	if err != nil {
		return fmt.Errorf("发送CRC16校验和失败: %v", err)
	}
	log.Printf("发送CRC16校验和: %x", crcBytes)

	// 发送结束标记（换行符）
	_, err = port.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("发送结束标记失败: %v", err)
	}
	log.Println("发送结束标记 (\\n)")

	return nil
}

func readFeedback(port *serial.Port, timeout time.Duration) (string, error) {
	feedback := make([]byte, 10)
	var totalRead int
	start := time.Now()

	for time.Since(start) < timeout {
		n, err := port.Read(feedback[totalRead:])
		if err != nil {
			return "", fmt.Errorf("读取反馈失败: %v", err)
		}
		totalRead += n
		if totalRead > 0 && (string(feedback[:totalRead]) == "OK" || string(feedback[:totalRead]) == "RETRY") {
			return string(feedback[:totalRead]), nil
		}
		time.Sleep(10 * time.Millisecond) // 防止CPU过载
	}

	return "", fmt.Errorf("反馈读取超时 (%v)", timeout)
}

func main() {
	// 定义原始消息
	message := Message{
		APIVersion:    "v3",
		ReceivedTopic: "",
		CorrelationID: "78f0dd39-5e0b-4002-809d-9bae380dfec3",
		RequestID:     "",
		ErrorCode:     0,
		Payload:       `eyJhcGlWZXJzaW9uIjoidjMiLCJyZXF1ZXN0SWQiOiI5YWQyOGM0Yi1iYTBkLTRjZWYtOTJhZC04ZTQxOGVjY2VkY2EiLCJldmVudCI6eyJhcGlWZXJzaW9uIjoidjMiLCJpZCI6IjAwOGY4YTMxLWUxOGUtNDkxYi05MTAwLTg5ZDI2YWZhNmJiYiIsImRldmljZU5hbWUiOiJSYW5kb20tSW50ZWdlci1EZXZpY2UiLCJwcm9maWxlTmFtZSI6IlJhbmRvbS1JbnRlZ2VyLURldmljZSIsInNvdXJjZU5hbWUiOiJJbnQ4Iiwib3JpZ2luIjoxNzQ4NDAxMzAzMzUwNjgwMjk1LCJyZWFkaW5ncyI6W3siaWQiOiJkYWM5NGQzMi0wODFiLTQ3NDMtYWQ1Zi00YmIwOGI1ODA0OTciLCJvcmlnaW4iOjE3NDg0MDEzMDMzNTA2ODAyOTUsImRldmljZU5hbWUiOiJSYW5kb20tSW50ZWdlci1EZXZpY2UiLCJyZXNvdXJjZU5hbWUiOiJJbnQ4IiwicHJvZmlsZU5hbWUiOiJSYW5kb20tSW50ZWdlci1EZXZpY2UiLCJ2YWx1ZVR5cGUiOiJJbnQ4IiwidmFsdWUiOiItNjMifV19fQ==`,
		ContentType:   "application/json",
	}

	// 序列化消息为JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("序列化消息失败: %v", err)
	}
	log.Printf("序列化后的JSON数据: %s", string(data))

	// 配置串口1
	config := &serial.Config{
		Name:        "COM6", // 替换为你的串口1名称
		Baud:        115200,
		Parity:      serial.ParityNone,
		ReadTimeout: 500 * time.Millisecond, // 设置默认读取超时
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	// 清空串口缓冲区
	port.Flush()

	// 发送数据并监听重传请求
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt == maxRetries {
			log.Fatalf("达到最大重试次数 (%d)，发送失败", maxRetries)
		}
		log.Printf("尝试发送数据 (第%d/%d次)", attempt, maxRetries)
		err = sendData(port, data)
		if err != nil {
			log.Fatalf("发送数据失败: %v", err)
		}

		// 监听接收端的反馈（等待3秒）
		feedback, err := readFeedback(port, 3*time.Second)
		if err != nil {
			log.Printf("读取反馈失败: %v", err)
			port.Flush() // 清空缓冲区以避免残留数据
			continue
		}
		log.Printf("接收到反馈: %q (字节数: %d)", feedback, len(feedback))

		if feedback == "OK" {
			log.Println("数据发送成功，收到确认")
			break
		} else if feedback == "RETRY" {
			log.Printf("接收端请求重传，尝试第%d次", attempt+1)
			port.Flush() // 清空缓冲区以避免残留数据
			continue
		} else {
			log.Printf("收到未知反馈: %q，尝试第%d次", feedback, attempt+1)
			port.Flush() // 清空缓冲区以避免残留数据
			continue
		}

	}

	log.Println("所有数据发送完成")
}
