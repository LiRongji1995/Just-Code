package main

import "fmt"

func main() {
	jobs := make(chan int, 5) // 创建一个容量为 5 的缓冲通道，用于发送任务编号（int 类型）
	done := make(chan bool)   // 创建一个无缓冲通道，用于通知主线程任务已完成
	go func() {
		for {
			j, more := <-jobs // 从 jobs 通道接收一个任务和“是否还有值”的标志
			if more {
				fmt.Println("received job", j) // 若通道未关闭，打印接收到的任务编号
			} else {
				fmt.Println("received all jobs") // 若通道已关闭，说明任务接收完毕
				done <- true                     // 向主线程发送通知，表示可以退出程序
				return                           // 终止 goroutine
			}
		}
	}()
	for j := 0; j < 3; j++ {
		jobs <- j                  // 向通道发送任务编号 j（0、1、2）
		fmt.Println("sent job", j) // 打印发送日志，观察主线程发送顺序
	}
	close(jobs)                  // 显式关闭通道，通知接收方不会再有新数据
	fmt.Println("sent all jobs") // 打印所有任务已发送完毕
	<-done                       // 主线程阻塞等待 done 通道中的通知，确保 worker goroutine 已完成
}
