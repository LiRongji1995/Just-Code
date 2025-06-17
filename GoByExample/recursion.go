package main

import "fmt" // 导入 fmt 包用于打印输出

// 定义一个递归函数 fact，用于计算阶乘
func fact(n int) int {
	if n == 0 {
		return 1 // 递归终止条件：0 的阶乘定义为 1
	}
	return n * fact(n-1) // 否则返回 n × (n-1)!
}

func main() {
	// 调用 fact(7)，即计算 7!（7 的阶乘），并打印结果
	fmt.Println(fact(7)) // 输出：5040

	// 声明一个变量 fib，它是一个函数类型：接收 int 返回 int
	var fib func(n int) int

	// 给 fib 赋值为一个匿名函数，这个匿名函数内部会递归调用 fib 本身
	fib = func(n int) int {
		if n < 2 {
			return n // 基础情况：f(0)=0，f(1)=1
		}
		return fib(n-1) + fib(n-2) // 否则递归地返回 f(n-1) + f(n-2)
	}

	// 调用 fib(7)，计算斐波那契数列中第 7 项（从 0 开始计数），并打印
	fmt.Println(fib(7)) // 输出：13（即 f(7) = 13）
}
