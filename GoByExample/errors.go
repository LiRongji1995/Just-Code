package main

import (
	"errors"
	"fmt"
)

// f1 是一个函数，接受一个 int 参数，返回 int 和 error 两个值
func f1(arg int) (int, error) {
	if arg == 42 { // 如果参数是 42，返回一个错误
		return -1, errors.New("can't work with 42") // 标准错误构造方式
	}
	return arg + 3, nil // 正常情况下返回值加 3，错误为 nil
}

// 自定义错误类型 argError，包含两个字段：参数值和问题描述
type argError struct {
	arg  int    // 出错的参数值
	prob string // 问题描述
}

// 为 argError 实现 error 接口的 Error() 方法
func (e *argError) Error() string {
	return fmt.Sprintf("%d - %s", e.arg, e.prob) // 返回错误描述字符串
}

// f2 是另一个函数，使用自定义错误类型进行返回
func f2(arg int) (int, error) {
	if arg == 42 { // 同样检查参数是否是 42
		return -1, &argError{arg, "can't work with it"} // 返回自定义错误类型的指针
	}
	return arg + 3, nil
}

func main() {
	// 测试 f1 函数对不同输入的表现
	for _, i := range []int{7, 42} {
		if r, e := f1(i); e != nil {
			fmt.Println("f1 failed:", e) // 如果返回 error，打印失败信息
		} else {
			fmt.Println("f1 worked:", r) // 否则打印结果
		}
	}

	// 测试 f2 函数对不同输入的表现
	for _, i := range []int{7, 42} {
		if r, e := f2(i); e != nil {
			fmt.Println("f2 failed:", e) // 打印自定义错误类型的描述（自动调用 Error() 方法）
		} else {
			fmt.Println("f2 worked:", r)
		}
	}

	// 进一步处理 f2 返回的错误，进行类型断言
	_, e := f2(42)
	if ae, ok := e.(*argError); ok { // 类型断言为 *argError
		fmt.Println(ae.arg)  // 输出出错的参数值
		fmt.Println(ae.prob) // 输出问题描述
	}
}
