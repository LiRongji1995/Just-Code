package main

//
//import (
//	"fmt"  // 导入格式化输出功能
//	"time" // 导入 time 包，用于模拟耗时操作
//)
//
//// 定义 worker 函数，接收一个 bool 类型的通道 done
//// 这个函数将在独立的 goroutine 中运行
//func worker(done chan bool) {
//	fmt.Println("working...") // 模拟开始工作
//	time.Sleep(time.Second)   // 停顿 1 秒钟，模拟工作过程
//	fmt.Println("done")       // 模拟工作完成
//
//	done <- true // 向通道发送 true，通知 main 函数“任务完成”
//}
//
//func main() {
//	done := make(chan bool, 1) // 创建一个容量为 1 的布尔通道，用于通知
//	go worker(done)            // 启动 worker goroutine，传入通道
//
//	<-done // 主 goroutine 在此等待通道中接收数据（阻塞）
//	// 当收到 worker 发来的 true 时继续执行，相当于“等待任务完成”
//}
