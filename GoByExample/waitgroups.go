package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int) {
	fmt.Printf("Worker %d starting", id)

	time.Sleep(time.Second)
	fmt.Printf("Worker %d done", id)
}
func main() {
	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1)

		i := i

		go func() {
			defer wg.Done()
			worker(i)
		}()
	}
	wg.Wait()
}

/*
时间轴图解：启动 5 个并发 worker，使用 WaitGroup 等待所有任务完成

主 goroutine                     worker 1           worker 2           worker 3           worker 4           worker 5
    |                                |                   |                   |                   |                   |
    | for 循环启动 goroutine -------->| 启动              |                   |                   |                   |
    | wg.Add(1)                      |                   |                   |                   |                   |
    |                                | 打印: starting     |                   |                   |                   |
    |--------------------------------|------------------>| 启动              |                   |                   |
    |                                |                   | 打印: starting     |                   |                   |
    |--------------------------------------------------->|------------------>| 启动              |                   |
    |                                |                   |                   | 打印: starting     |                   |
    |--------------------------------------------------------------------->|------------------>| 启动              |
    |                                |                   |                   |                   | 打印: starting     |
    | wg.Wait()                      |                   |                   |                   |                   |
    |                                | 工作中（Sleep）    | 工作中（Sleep）    | 工作中（Sleep）    | 工作中（Sleep）    | 工作中（Sleep）
    |                                | 完成，打印: done    |                   |                   |                   |
    |                                | wg.Done() -------->|                   |                   |                   |
    |                                |                   | 完成，打印: done    |                   |                   |
    |                                |                   | wg.Done() -------->|                   |                   |
    |                                |                   |                   | 完成，打印: done    |                   |
    |                                |                   |                   | wg.Done() -------->|                   |
    |                                |                   |                   |                   | 完成，打印: done    |
    |                                |                   |                   |                   | wg.Done() -------->|
    |                                |                   |                   |                   |                   | 完成，打印: done
    |                                |                   |                   |                   |                   | wg.Done() ------>
    | 所有 worker 执行完毕           |                   |                   |                   |                   |
    | wg.Wait() 返回，程序结束       |                   |                   |                   |                   |
*/
