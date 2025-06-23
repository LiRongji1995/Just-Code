package main // 声明 main 包，程序入口

import (
	"fmt"         // 导入用于格式化输出的标准库
	"sync"        // 导入 sync 包，用于并发控制（WaitGroup）
	"sync/atomic" // 导入 atomic 包，用于原子操作（线程安全）
)

func main() {
	var ops uint64 // 定义一个 uint64 类型的共享变量，用于统计操作次数

	var wg sync.WaitGroup // 声明 WaitGroup，用于等待所有 goroutine 完成

	// 启动 50 个并发 goroutine，每个 goroutine 执行 1000 次加操作
	for i := 0; i < 50; i++ {
		wg.Add(1) // 注册一个 goroutine 的计数

		go func() {
			// 每个 goroutine 对共享变量 ops 执行 1000 次原子递增
			for c := 0; c < 1000; c++ {
				//这是一个原子操作，它通过 CPU 提供的低层原子指令（如 x86 的 LOCK 前缀），保证对变量 ops 的加法在机器指令层面不可中断。
				atomic.AddUint64(&ops, 1) // 原子操作，确保线程安全
			}
			wg.Done() // 当前 goroutine 完成后调用 Done，减少 WaitGroup 计数
		}()
	}

	wg.Wait() // 主 goroutine 阻塞等待所有 50 个 goroutine 完成

	fmt.Println("ops:", ops) // 输出最终计数结果，预期为 50 * 1000 = 50000
}
