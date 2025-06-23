package main

import (
	"fmt"
	"time"
)

func main() {

	requests := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		requests <- i
	}
	close(requests)

	limiter := time.Tick(200 * time.Millisecond)

	for req := range requests {
		<-limiter
		fmt.Println("request", req, time.Now())
	}

	burstyLimiter := make(chan time.Time, 3)

	for i := 0; i < 3; i++ {
		burstyLimiter <- time.Now()
	}

	go func() {
		for t := range time.Tick(200 * time.Millisecond) {
			burstyLimiter <- t
		}
	}()

	burstyRequests := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		burstyRequests <- i
	}
	close(burstyRequests)
	for req := range burstyRequests {
		<-burstyLimiter
		fmt.Println("request", req, time.Now())
	}
}

/*
时间轴图解：分为两段
第一段：limiter 每 200ms 处理一个请求（固定速率）
第二段：burstyLimiter 先突发处理 3 个请求，再按 200ms 逐个补充令牌

主 goroutine                          limiter                           bursty goroutine

    |                                     |                                    |
    | 创建 requests chan 并填入5个任务    |                                    |
    | 创建 limiter := time.Tick(200ms)    | 每200ms 生成1个tick                |
    |------------------------------------>|----------------------------------> |
    | 读取 req 1 <- requests              |                                    |
    | <-limiter 阻塞直到tick              | 200ms → 发tick ------------------->|
    | 打印 req 1 time                     |                                    |
    | 读取 req 2                          |                                    |
    | <-limiter 阻塞直到tick              | 400ms → 发tick ------------------->|
    | 打印 req 2                          |                                    |
    | 读取 req 3                          |                                    |
    | <-limiter 阻塞直到tick              | 600ms → 发tick ------------------->|
    | 打印 req 3                          |                                    |
    | 读取 req 4                          |                                    |
    | <-limiter 阻塞直到tick              | 800ms → 发tick ------------------->|
    | 打印 req 4                          |                                    |
    | 读取 req 5                          |                                    |
    | <-limiter 阻塞直到tick              | 1000ms → 发tick ------------------>|
    | 打印 req 5                          |                                    |
    |                                     |                                    |
    | 创建 burstyLimiter chan (cap=3)     |                                    |
    | 初始填充3个令牌 ------------------->|                                    |
    | 启动 goroutine 每200ms补充一个令牌  |                                    |
    |                                     | ticker(200ms) → 准备发送令牌      |
    | 创建 burstyRequests chan 填5个任务 |                                    |
    | 读取 req 1 <- burstyRequests        |                                    |
    | <-burstyLimiter（立即有）           |                                    |
    | 打印 req 1                          |                                    |
    | 读取 req 2                          |                                    |
    | <-burstyLimiter（立即有）           |                                    |
    | 打印 req 2                          |                                    |
    | 读取 req 3                          |                                    |
    | <-burstyLimiter（立即有）           |                                    |
    | 打印 req 3                          |                                    |
    | 读取 req 4                          |                                    |
    | <-burstyLimiter（此时为空，等待）   | 200ms → goroutine发送一个令牌 --->|
    | 打印 req 4                          |                                    |
    | 读取 req 5                          |                                    |
    | <-burstyLimiter（再等待）           | 400ms → goroutine发送一个令牌 --->|
    | 打印 req 5                          |                                    |
    | 所有请求处理完毕，程序退出          |                                    |
*/
