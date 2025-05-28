// package main

// import (
// 	"log"
// 	"serialcomm/serialcomm"
// 	"time"
// )

// func main() {
// 	receiver, err := serialcomm.NewSerialReceiver(&serialcomm.SerialConfig{
// 		PortName:    "COM6",
// 		BaudRate:    115200,
// 		ReadTimeout: 500 * time.Millisecond,
// 		MaxLength:   10000,
// 		ReadCallback: func(msg *serialcomm.Message, payload *serialcomm.Payload) {
// 			log.Printf("收到消息: %+v", msg)
// 			log.Printf("Payload 内容: %+v", payload)
// 		},
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	err = receiver.Start()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	select {} // 持续运行
// }