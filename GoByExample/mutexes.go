package main // 声明 main 包，程序入口

import (
	"fmt"  // 导入 fmt 包，用于输出打印
	"sync" // 导入 sync 包，用于并发控制（Mutex、WaitGroup）
)

// Container 是一个结构体，封装了一个互斥锁和一个字符串计数器 map
type Container struct {
	mu       sync.Mutex     // 互斥锁，用于保护 counters 的并发访问
	counters map[string]int // 保存计数数据，例如 {"a": 123, "b": 456}
}

// inc 方法是对指定 name 的计数器加 1，使用互斥锁保护
func (c *Container) inc(name string) {
	c.mu.Lock()         // 加锁，防止并发写入冲突
	defer c.mu.Unlock() // 函数退出前自动释放锁
	c.counters[name]++  // 执行加 1 操作（临界区）
}

func main() {
	c := Container{
		counters: map[string]int{"a": 0, "b": 0}, // 初始化计数器 map
	}

	var wg sync.WaitGroup // 用于等待所有 goroutine 执行完毕

	// 定义一个函数 doIncrement，用于对指定计数器执行 n 次自增
	doIncrement := func(name string, n int) {
		for i := 0; i < n; i++ {
			c.inc(name) // 并发安全地对 map 中的 name 执行加 1
		}
		wg.Done() // 当前 goroutine 完成后调用 Done()
	}

	// 启动 3 个 goroutine
	// 其中两个同时对 "a" 进行 10000 次自增，另一个对 "b" 进行 10000 次自增
	wg.Add(3)                  // 登记 3 个任务
	go doIncrement("a", 10000) // goroutine 1
	go doIncrement("a", 10000) // goroutine 2（也对 "a"）
	go doIncrement("b", 10000) // goroutine 3

	wg.Wait() // 等待所有 goroutine 执行完毕

	fmt.Println(c.counters) // 输出最终的计数结果，例如：map[a:20000 b:10000]
}
