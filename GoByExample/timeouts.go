package main

import (
	"fmt"  // 引入 fmt 包用于输出
	"time" // 引入 time 包用于延时和超时控制
)

func main() {

	c1 := make(chan string, 1) // 创建一个带缓冲区大小为1的 string 通道 c1
	go func() {
		time.Sleep(2 * time.Second) // 模拟耗时操作：睡眠 2 秒
		c1 <- "result 1"            // 发送字符串 "result 1" 到 c1
	}()

	// 使用 select 等待通道 c1 的返回或超时
	select {
	case res := <-c1: // 如果 c1 在 1 秒内有值可读
		fmt.Println(res) // 打印读取到的值
	case <-time.After(1 * time.Second): // 如果 1 秒内没收到消息
		fmt.Println("timeout") // 打印 "timeout"
	}

	c2 := make(chan string, 1) // 创建另一个带缓冲区的通道 c2
	go func() {
		time.Sleep(2 * time.Second) // 睡眠 2 秒模拟操作
		c2 <- "result 2"            // 发送字符串 "result 2" 到 c2
	}()

	// 第二次使用 select，这次等待时间为 3 秒
	select {
	case res := <-c2: // 如果 c2 在 3 秒内有值可读
		fmt.Println(res) // 打印读取到的值
	case <-time.After(3 * time.Second): // 如果超时
		fmt.Println("timeout 2") // 打印 "timeout 2"
	}
}
