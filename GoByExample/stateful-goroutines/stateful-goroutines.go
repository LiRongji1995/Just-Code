package main

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

// 定义一个读取操作的结构体
type readOp struct {
	key  int      // 要读取的键
	resp chan int // 响应读取结果的通道
}

// 定义一个写入操作的结构体
type writeOp struct {
	key  int       // 要写入的键
	val  int       // 写入的值
	resp chan bool // 通知写入成功的通道
}

func main() {
	var readOps uint64  // 用于统计总的读取操作数（使用原子操作）
	var writeOps uint64 // 用于统计总的写入操作数（使用原子操作）

	reads := make(chan readOp)   // 所有读取请求发送到这个通道
	writes := make(chan writeOp) // 所有写入请求发送到这个通道
	go func() {
		var state = make(map[int]int) // 共享状态，只能通过这个 goroutine 修改
		for {
			select {
			case read := <-reads: // 收到读取请求
				read.resp <- state[read.key] // 读取结果写回响应通道
			case write := <-writes: // 收到写入请求
				state[write.key] = write.val // 写入 map
				write.resp <- true           // 通知写入完成
			}
		}
	}()
	for r := 0; r < 100; r++ {
		go func() {
			for {
				read := readOp{
					key:  rand.Intn(5),   // 随机读取 key（0~4）
					resp: make(chan int)} // 创建响应通道
				reads <- read                 // 把读取请求发给共享状态 goroutine
				<-read.resp                   // 等待读取结果（阻塞直到收到）
				atomic.AddUint64(&readOps, 1) // 统计读操作 +1（原子）
				time.Sleep(time.Millisecond)  // 控制读频率
			}
		}()
	}
	for w := 0; w < 10; w++ {
		go func() {
			for {
				write := writeOp{
					key:  rand.Intn(5),    // 随机写入 key（0~4）
					val:  rand.Intn(100),  // 随机写入值
					resp: make(chan bool)} // 创建响应通道
				writes <- write                // 发出写入请求
				<-write.resp                   // 等待确认写入完成
				atomic.AddUint64(&writeOps, 1) // 统计写操作 +1（原子）
				time.Sleep(time.Millisecond)   // 控制写频率
			}
		}()
	}
	time.Sleep(time.Second) // 主线程休眠 1 秒，让 goroutines 执行

	readOpsFinal := atomic.LoadUint64(&readOps) // 获取最终读取次数
	fmt.Println("readOps:", readOpsFinal)

	writeOpsFinal := atomic.LoadUint64(&writeOps) // 获取最终写入次数
	fmt.Println("writeOps:", writeOpsFinal)
}

// 总结：使用 goroutine + channel 来协调对共享 map 的访问
// 优点：避免了使用传统的互斥锁（sync.Mutex），让数据访问“被动式”地调度，更清晰
// 缺点：结构更复杂，尤其是当程序需要和多个通道交互时，代码会变得更难维护

// 对比传统互斥锁方式：
//   - 传统方式：多个 goroutine 并发访问 map，需要加锁（容易出错，死锁风险）
//   - 本例方式：将 map 封装进一个 goroutine，通过通道处理所有请求，避免显式加锁

// 适用场景：
//   ✅ 学习理解协程间通信方式
//   ✅ 对并发安全要求高，且写逻辑简单明确时
//   ❌ 若涉及多个 map、多个通道、复杂状态协作，会变得非常难以维护

// 建议：初学者应优先使用结构直观的方式（如 sync.Mutex），理解清楚并发行为后再尝试 channel 协调式设计
