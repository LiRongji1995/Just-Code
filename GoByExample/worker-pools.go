package main // 声明 main 包，程序的入口

import (
	"fmt"
	"time"
)

// worker 是工作池中的一个工人，负责从任务通道中拉取任务并处理。
// id: 工人编号，用于区分不同的 goroutine 输出
// jobs: 只读通道，用于接收任务编号
// results: 只写通道，用于发送任务处理结果
func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs { // 持续从任务队列中获取任务，直到通道被关闭
		fmt.Println("worker", id, "started job", j)
		time.Sleep(time.Second) // 模拟任务耗时 1 秒
		fmt.Println("worker", id, "finished job", j)
		results <- j * 2 // 处理结果通过 results 通道发送回主线程
	}
}

func main() {
	const numJobs = 5                  // 任务总数（比如我们要处理5个任务）
	jobs := make(chan int, numJobs)    // 任务通道，容量为任务总数
	results := make(chan int, numJobs) // 结果通道，容量同样为任务总数

	// 启动 3 个 worker，这些 goroutine 组成一个“工作池”，共享任务通道
	// 每个 worker 独立执行，处理从 jobs 通道中读取的任务
	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results)
	}

	// 主线程将所有任务编号写入 jobs 通道，作为“投喂任务”
	// 在实际场景中这些可能是任务 ID、结构体、函数等
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs) // 所有任务发送完毕，关闭任务通道，通知 worker 无更多任务

	// 主线程从 results 通道中收集每个任务的处理结果
	// 注意：这一步阻塞直到所有结果都被收集完成，确保程序不提前退出
	for a := 1; a <= numJobs; a++ {
		<-results // 这里只是等待完成，可以用变量接收结果进行打印或处理
	}

	// 至此，所有任务完成，主程序退出
}

/*
时间轴图解：3 个 worker 并发处理 5 个任务，每个任务耗时 1 秒

主 goroutine                          worker 1                   worker 2                   worker 3
    |                                     |                          |                          |
    | 创建 jobs / results 通道            |                          |                          |
    | 启动 3 个 worker ------------------> | 等待任务                |                          |
    |------------------------------------>|------------------------->| 等待任务                |
    |------------------------------------>|------------------------->|------------------------->| 等待任务
    | 投递 job 1 ------------------------>| 处理 job 1              |                          |
    | 投递 job 2 ------------------------>|                          | 处理 job 2              |
    | 投递 job 3 ------------------------>|                          |                          | 处理 job 3
    | 投递 job 4                          |                          |                          |
    | 投递 job 5                          |                          |                          |
    | 关闭 jobs 通道                     |                          |                          |
    |                                     | 完成 job 1              |                          |
    |                                     | 写结果 1 到 results      |                          |
    |                                     | 拉取 job 4 ------------>|                          |
    |                                     | 开始处理 job 4          |                          |
    |                                     |                          | 完成 job 2              |
    |                                     |                          | 写结果 2 到 results      |
    |                                     |                          | 拉取 job 5 ------------>|
    |                                     |                          | 开始处理 job 5          |
    |                                     |                          |                          | 完成 job 3
    |                                     |                          |                          | 写结果 3 到 results
    |                                     | 完成 job 4              |                          |
    |                                     | 写结果 4 到 results      |                          |
    |                                     |                          | 完成 job 5              |
    |                                     |                          | 写结果 5 到 results      |
    | 从 results 接收 5 个结果           |                          |                          |
    | 程序结束                            | 所有 worker 退出（jobs 被关闭）
*/
