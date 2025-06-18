package main

import (
	"fmt"  // 导入 fmt 包，用于格式化输出
	"time" // 导入 time 包，用于睡眠暂停程序
)

func f(from string) {
	// 定义一个函数 f，参数为字符串类型的 from
	for i := 0; i < 3; i++ {
		// 循环三次，输出 from 和当前迭代的 i 值
		fmt.Println(from, ":", i)
	}
}

func main() {

	f("direct")
	// 直接调用 f("direct")，在主 goroutine 中顺序执行

	go f("goroutine")
	// 启动一个新的 goroutine 来运行 f("goroutine")
	// 这意味着该函数将在后台异步执行，不阻塞主函数

	go func(msg string) {
		// 定义并立即启动一个匿名 goroutine 函数，接收 msg 参数
		fmt.Println(msg)
	}("going")
	// 将 "going" 作为参数传入匿名函数中

	time.Sleep(time.Second)
	// 为了让上面的两个 goroutine 有时间完成执行，主函数休眠 1 秒
	// 否则程序可能在 goroutine 输出前就退出了

	fmt.Println("done")
	// 主 goroutine 输出 "done"
}
