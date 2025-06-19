package main

import (
	"fmt"
	"time"
)

func main() {
	// 启动一个每秒滴答的 ticker
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop() // 程序结束时停止 ticker，防止资源泄露

	// 启动一个 goroutine 来打印每秒的 “tiktok”
	go func() {
		for t := range ticker.C {
			fmt.Println("tiktok", t.Format("15:04:05"))
		}
	}()
	/*
	   时间轴图解：timer1 正常触发，timer2 被提前停止

	   主 goroutine                     goroutine（用于timer2）
	       |                                 |
	       | timer1 := time.NewTimer(2s)     |
	       | <-timer1.C                      |
	       |（阻塞等待2秒）              	     |
	       |-------------------------------->|
	       |                                 |
	       | 打印: Timer 1 fired   		     |
	       |                                 |
	       | timer2 := time.NewTimer(3s)     |
	       | go func() { <-timer2.C ... }    |
	       |-------------------------------> | goroutine 启动但尚未调度执行
	       |                                 |
	       | stop2 := timer2.Stop()          |
	       | stop2 == true                   |
	       | 打印: Timer 2 stopped	         |
	       |                                 |
	       | time.Sleep(2 * time.Second)     |
	       |-------------------------------->|
	       |                                 | goroutine 执行到 <-timer2.C
	       |                                 | 由于 timer2 已停止，
	       |                                 | <-timer2.C 永久阻塞，无法打印
	*/

	// 原有 timer1 逻辑
	timer1 := time.NewTimer(2 * time.Second)
	<-timer1.C
	fmt.Println("Timer 1 fired")

	// 原有 timer2 逻辑
	timer2 := time.NewTimer(3 * time.Second)
	go func() {
		<-timer2.C
		fmt.Println("Timer 2 fired")
	}()
	stop2 := timer2.Stop()
	if stop2 {
		fmt.Println("Timer 2 stopped")
	}

	time.Sleep(2 * time.Second) // 确保所有输出都完成
}
