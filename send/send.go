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
	return crc16.Checksum(data, &crc16.Table{})
}

func sendData(port *serial.Port, data []byte) error {
	// 添加4字节长度前缀（大端序）
	length := uint32(len(data))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	_, err := port.Write(lengthBytes)
	if err != nil {
		return fmt.Errorf("发送长度前缀失败: %v", err)
	}
	log.Printf("发送长度前缀: %d字节，内容: %x", len(lengthBytes), lengthBytes)

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
		Name:        "COM6", // 替换为你的串口1名称，例如 /dev/ttyS0 (Linux) 或 COM1 (Windows)
		Baud:        115200,
		Parity:      serial.ParityNone,
		ReadTimeout: 1 * time.Second,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	// 发送数据并监听重传请求
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("尝试发送数据 (第%d/%d次)", attempt, maxRetries)
		err = sendData(port, data)
		if err != nil {
			log.Fatalf("发送数据失败: %v", err)
		}

		// 监听接收端的反馈（等待1秒）
		feedback := make([]byte, 10)
		n, err := port.Read(feedback)
		if err != nil {
			log.Printf("读取反馈失败: %v", err)
			continue
		}
		feedbackStr := string(feedback[:n])
		log.Printf("接收到反馈: %q", feedbackStr)

		if feedbackStr == "OK" {
			log.Println("数据发送成功，收到确认")
			break
		} else if feedbackStr == "RETRY" {
			log.Printf("接收端请求重传，尝试第%d次", attempt+1)
			continue
		}

		if attempt == maxRetries {
			log.Fatalf("达到最大重试次数 (%d)，发送失败", maxRetries)
		}
	}

	log.Println("所有数据发送完成")
}
