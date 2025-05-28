package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"strings"
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

func main() {
	// 配置串口2
	config := &serial.Config{
		Name:        "/dev/ttyUSB0", // 替换为你的串口2名称，例如 /dev/ttyS1 (Linux) 或 COM2 (Windows)
		Baud:        115200,
		Parity:      serial.ParityNone,
		ReadTimeout: 500 * time.Millisecond,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	log.Println("开始监听串口...")

	// 缓冲区和读取逻辑
	var buffer bytes.Buffer
	data := make([]byte, 1024)
	var expectedLength uint32
	lastDataTime := time.Now()
	timeout := 5 * time.Second // 超时时间

	for {
		// 读取串口数据
		n, err := port.Read(data)
		if err != nil {
			log.Printf("读取串口数据失败: %v", err)
			continue
		}
		if n == 0 {
			// 检查超时
			if time.Since(lastDataTime) > timeout && buffer.Len() > 0 {
				log.Printf("接收超时，清空缓冲区（大小: %d）", buffer.Len())
				buffer.Reset()
				expectedLength = 0
			}
			continue
		}

		// 更新最后接收时间
		lastDataTime = time.Now()

		// 过滤非ASCII字符（只保留32-126和换行符10）
		cleanData := make([]byte, 0, n)
		for _, b := range data[:n] {
			if (b >= 32 && b <= 126) || b == 10 {
				cleanData = append(cleanData, b)
			}
		}
		buffer.Write(cleanData)
		log.Printf("接收到%d字节，清理后%d字节，缓冲区大小: %d，内容: %q (十六进制: %x)", n, len(cleanData), buffer.Len(), cleanData, cleanData)

		// 读取长度前缀
		if expectedLength == 0 && buffer.Len() >= 4 {
			lengthBytes := buffer.Next(4)
			expectedLength = binary.BigEndian.Uint32(lengthBytes)
			log.Printf("读取到长度前缀: %d字节", expectedLength)
		}

		// 检查是否收到完整数据包（长度+2字节CRC+换行符）
		if expectedLength > 0 && buffer.Len() >= int(expectedLength)+2 && strings.Contains(buffer.String(), "\n") {
			// 提取数据和CRC
			dataPacket := buffer.Next(int(expectedLength))
			crcBytes := buffer.Next(2)
			receivedCRC := binary.BigEndian.Uint16(crcBytes)
			calculatedCRC := calculateCRC16(dataPacket)

			// 验证CRC
			if receivedCRC != calculatedCRC {
				log.Printf("CRC校验失败，接收到的CRC: %x，计算的CRC: %x", receivedCRC, calculatedCRC)
				// 发送重传请求
				_, err := port.Write([]byte("RETRY"))
				if err != nil {
					log.Printf("发送重传请求失败: %v", err)
				} else {
					log.Println("发送重传请求: RETRY")
				}
				buffer.Reset()
				expectedLength = 0
				continue
			}

			// 确保数据包以换行符结束
			_, err := buffer.ReadBytes('\n')
			if err != nil {
				log.Printf("未找到结束标记，等待更多数据")
				continue
			}

			// 尝试解析JSON
			var message Message
			err = json.Unmarshal(dataPacket, &message)
			if err != nil {
				log.Printf("JSON解析失败: %v, 数据: %q (十六进制: %x)", err, dataPacket, dataPacket)
				// 发送重传请求
				_, err := port.Write([]byte("RETRY"))
				if err != nil {
					log.Printf("发送重传请求失败: %v", err)
				} else {
					log.Println("发送重传请求: RETRY")
				}
				buffer.Reset()
				expectedLength = 0
				continue
			}

			// 成功解析，发送确认
			_, err = port.Write([]byte("OK"))
			if err != nil {
				log.Printf("发送确认失败: %v", err)
			} else {
				log.Println("发送确认: OK")
			}

			// 打印消息
			log.Printf("接收并解析消息: %+v\n", message)

			// 重置状态
			buffer.Reset()
			expectedLength = 0
		}

		// 防止CPU过载
		time.Sleep(10 * time.Millisecond)
	}
}
