package main

import "fmt"

// intSeq 是一个返回“函数”的函数，返回值类型是 func() int（即一个无参数、返回 int 的函数）
func intSeq() func() int {
	i := 0 // 定义局部变量 i，用来记录计数状态

	// 返回一个匿名函数（闭包）。它引用了 i，并在每次调用时执行 i++ 并返回 i
	return func() int {
		i++
		return i
	}
}

func main() {
	// nextInt 是一个闭包，拥有自己的 i 状态（初始为 0）
	nextInt := intSeq()

	// 每次调用 nextInt()，i 自增一次，状态在闭包中持续存在
	fmt.Println(nextInt()) // 输出 1
	fmt.Println(nextInt()) // 输出 2
	fmt.Println(nextInt()) // 输出 3

	// newInts 是另一个全新的闭包，拥有独立的 i（与 nextInt 的 i 无关）
	newInts := intSeq()
	fmt.Println(newInts()) // 输出 1，因为它自己的 i 是 0 开始
}
