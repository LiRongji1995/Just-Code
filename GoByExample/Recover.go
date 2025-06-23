package main // 声明 main 包，是 Go 程序的入口

import "fmt" // 导入 fmt 包，用于格式化输出

// mayPanic 是一个可能引发 panic 的函数
func mayPanic() {
	// panic 会立即执行以下操作：
	// 1. 停止当前函数的正常执行流程（后续代码不会执行）
	// 2. 开始回溯调用栈（stack unwinding），按后进先出顺序执行已注册的 defer 函数
	// 3. 如果没有 recover 捕获，最终会打印 panic 信息并终止程序
	panic("a problem") // 主动触发 panic，模拟程序遇到严重错误
}

func main() {
	// defer 注册一个延迟函数，在以下情况执行：
	// 1. 函数正常返回时
	// 2. 发生 panic 时（在栈展开过程中）
	defer func() {
		// 当 panic 发生时，运行时系统会执行这个 defer 函数
		if r := recover(); r != nil {
			// recover 机制的具体行为：
			// 1. 仅在 defer 函数中调用时才有效
			// 2. 会捕获最近的 panic 值并停止 panic 的传播
			// 3. 返回的 panic 值就是传递给 panic() 的参数
			fmt.Println("Recovered. Error:", r)

			// 关键点说明（为什么不会回到 panic 点）：
			// 1. Go 的 panic/recover 不是传统的 try-catch 机制
			// 2. panic 会导致当前函数的执行上下文被立即丢弃（包括局部变量、执行位置等）
			// 3. recover 只是阻止 panic 继续向上传播，但无法恢复已丢弃的执行上下文
			// 4. 设计哲学：panic 表示程序遇到无法继续执行的严重错误，应该终止当前任务流
		}
	}()

	// 调用可能触发 panic 的函数
	mayPanic()

	// 这行代码不会执行，具体原因：
	// 1. mayPanic() 中的 panic 导致 main 函数的执行帧（execution frame）被标记为"panicking"状态
	// 2. 运行时系统开始处理 panic 流程：
	//    a) 查找当前函数的 defer 函数（找到上面的 defer）
	//    b) 执行 defer 函数中的 recover 调用
	//    c) 由于 recover 成功捕获，停止 panic 传播
	// 3. 但是：
	//    a) panic 已导致 main 函数的正常执行流被永久中断
	//    b) 运行时不会维护足够的状态信息来恢复 panic 点的执行
	//    c) 按照 Go 的设计，panic 后的代码被认为处于不可靠状态，不应继续执行
	// 4. 因此程序会在 defer 执行完毕后，直接结束 main 函数的执行
	fmt.Println("After mayPanic()")

}
