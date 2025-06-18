package main

import (
	"fmt"  // 导入 fmt 包，用于打印输出
	"time" // 导入 time 包，用于添加延迟模拟异步任务
)

func main() {
	c1 := make(chan string) // 创建一个 string 类型的无缓冲通道 c1
	c2 := make(chan string) // 创建一个 string 类型的无缓冲通道 c2

	// 启动一个 goroutine，1 秒后向 c1 发送 "one"
	go func() {
		time.Sleep(1 * time.Second) // 模拟耗时操作
		c1 <- "one"                 // 向通道 c1 发送字符串
	}()

	// 启动另一个 goroutine，2 秒后向 c2 发送 "two"
	go func() {
		time.Sleep(2 * time.Second) // 模拟更长的耗时操作
		c2 <- "two"                 // 向通道 c2 发送字符串
	}()

	// 使用 select 语句等待两个通道中的消息
	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-c1: // 如果 c1 有消息就读取
			fmt.Println("received", msg1)
		case msg2 := <-c2: // 如果 c2 有消息就读取
			fmt.Println("received", msg2)
		}
	}
}
