package main // 声明 main 包，程序的入口

import (
	"fmt"  // 导入格式化输出库
	"time" // 导入时间处理库
)

func main() {
	ticker := time.NewTicker(500 * time.Millisecond) // 创建一个 ticker，每 500ms 触发一次
	done := make(chan bool)                          // 创建一个用于通知退出的布尔型通道

	go func() { // 启动一个 goroutine 处理 ticker 的定时事件
		for {
			select {
			case <-done: // 如果从 done 通道接收到退出信号
				return // 退出 goroutine
			case t := <-ticker.C: // 从 ticker 的通道中接收时间信号
				fmt.Println("Tick at", t) // 打印当前时间
			}
		}
	}()

	time.Sleep(1600 * time.Millisecond) // 主线程睡眠 1.6 秒，让 ticker 至少触发三次
	ticker.Stop()                       // 停止 ticker，防止继续触发
	done <- true                        // 向 done 通道发送信号，通知 goroutine 停止
	fmt.Println("Ticker stopped")       // 打印结束信息
}

/*
时间轴图解：ticker 每 500ms 触发一次，1.6 秒后停止，通知 goroutine 退出

主 goroutine                        goroutine（打印 Tick）

    |                                   |
    | ticker := time.NewTicker(500ms)   |
    | done := make(chan bool)           |
    | go func() { ... }                 |
    |---------------------------------> | goroutine 启动，进入 select 循环
    |                                   |
    |                                   | <-ticker.C 收到 1 次（约 0.5s）
    |                                   | 打印 Tick at T1
    |                                   |
    |                                   | <-ticker.C 收到 2 次（约 1.0s）
    |                                   | 打印 Tick at T2
    |                                   |
    |                                   | <-ticker.C 收到 3 次（约 1.5s）
    |                                   | 打印 Tick at T3
    |                                   |
    | time.Sleep(1600ms)                |
    | ticker.Stop()                     | 停止 ticker，通道不再有新值
    | done <- true                      |
    | 打印: Ticker stopped               |
    |---------------------------------> | goroutine 收到 <-done，退出 return
*/
