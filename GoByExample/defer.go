package main // 定义程序的包名为 main，表明这是可独立执行的程序入口

import (
	"fmt" // 导入 fmt 包用于格式化输入输出
	"os"  // 导入 os 包用于操作系统功能，如文件操作
)

func main() {
	// 调用 createFile 创建一个文件，并返回文件指针 f
	f := createFile("defer.txt")

	// 使用 defer 确保在 main 函数结束前自动调用 closeFile 关闭文件
	defer closeFile(f)

	// 写入内容到文件
	writeFile(f)
}

// createFile 创建一个名为 p 的文件，并返回对应的文件指针
func createFile(p string) *os.File {
	fmt.Println("creating") // 打印创建提示信息
	f, err := os.Create(p)  // 尝试创建文件
	if err != nil {
		panic(err) // 如果创建失败，抛出 panic 错误并终止程序
	}
	return f // 返回文件指针
}

// writeFile 向指定文件写入数据
func writeFile(f *os.File) {
	fmt.Println("writing")  // 打印写入提示信息
	fmt.Fprintln(f, "data") // 向文件写入一行 "data"
}

// closeFile 关闭指定文件并处理关闭失败的错误
func closeFile(f *os.File) {
	fmt.Println("closing") // 打印关闭提示信息
	err := f.Close()       // 尝试关闭文件
	if err != nil {
		// 如果关闭失败，向标准错误输出错误信息并退出程序
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
