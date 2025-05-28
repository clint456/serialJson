// main.go
package main

import (
	"log"
	"time"

	"serialcomm/serialcomm" // 替换为你的实际模块路径
)

func main() {
	receiver, err := serialcomm.NewSerialReceiver(&serialcomm.SerialConfig{
		PortName:    "COM7", // 或 "/dev/ttyUSB0"（Linux）
		BaudRate:    115200,
		ReadTimeout: 300 * time.Millisecond,
		MaxLength:   4096,
		ReadCallback: func(msg *serialcomm.Message, payload *serialcomm.Payload) {
			log.Println("✅ 收到消息:")
			log.Printf("  ▶ Message: %+v\n", msg)
			log.Printf("  ▶ Payload: %+v\n", payload)
		},
	})
	if err != nil {
		log.Fatalf("初始化串口失败: %v", err)
	}

	err = receiver.Start()
	if err != nil {
		log.Fatalf("启动串口监听失败: %v", err)
	}

	log.Println("🔌 正在监听串口数据... Ctrl+C 可退出")
	select {} // 无限阻塞主线程
}
