package main

import (
	"fmt"
	"unicode/utf8" // 引入 utf8 包，用于处理 Unicode 字符的编码与解码
)

func main() {
	const s = "สวัสดี" // 定义一个包含泰语字符的字符串（6 个字符，但占用多个字节）

	fmt.Println("Len:", len(s)) // 打印字符串的字节长度（UTF-8 编码），结果为 18

	// 用 for 循环按字节逐个输出字符串的每个字节（十六进制）
	for i := 0; i < len(s); i++ {
		fmt.Printf("%x ", s[i]) // 输出单个字节的 16 进制值
	}
	fmt.Println()

	// 使用 utf8 包计算字符串中的字符数量（rune 数），结果应为 6
	fmt.Println("Rune count:", utf8.RuneCountInString(s))

	// 使用 range 遍历字符串，range 会按字符（rune）自动处理 UTF-8 编码
	for idx, runeVal := range s {
		// 打印每个 Unicode 字符（如 U+0E2A）和它在字节序列中的起始位置
		fmt.Printf("%#U starts at %d\n", runeVal, idx)
	}

	fmt.Println("\nUsing DecodeRuneInString")
	// 使用 utf8.DecodeRuneInString 手动解码每个字符（rune）
	for i, w := 0, 0; i < len(s); i += w {
		// 从 s[i:] 开始解析一个 rune，并返回该 rune 及其字节宽度
		runeValue, width := utf8.DecodeRuneInString(s[i:])
		fmt.Printf("%#U starts at %d\n", runeValue, i)
		w = width // 将当前 rune 的宽度赋给 w，以便 i += w 移动到下一个字符

		// 调用辅助函数 examineRune，对特定字符进行判断
		examineRune(runeValue)
	}
}

// examineRune 接收一个 rune 参数并根据其值输出特定信息
func examineRune(r rune) {
	if r == 't' {
		fmt.Println("found tee") // 如果字符是 't'（英文），打印提示
	} else if r == 'ส' {
		fmt.Println("found so sua") // 如果字符是泰语字符 'ส'，打印提示
	}
}
