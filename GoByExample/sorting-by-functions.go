package main

import (
	"fmt"  // 用于打印输出
	"sort" // Go 标准库中的排序工具
)

// 定义一个新的类型 byLength，它是 string 切片的别名
type byLength []string

// 实现 sort.Interface 接口的三个方法：Len, Swap, Less

// Len 返回切片的长度
func (s byLength) Len() int {
	return len(s)
}

// Swap 交换切片中索引 i 和 j 的元素
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less 用于排序比较：如果第 i 个元素的长度小于第 j 个元素，就返回 true
func (s byLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func main() {
	// 创建一个字符串切片
	fruits := []string{"peach", "banana", "kiwi"}

	// 将 fruits（类型是 []string）强制转换成 byLength 类型
	//使用 sort.Sort 按照自定义的规则对 fruits 排序（按字符串长度升序）
	sort.Sort(byLength(fruits)) //在 Go 中，任何类型（包括基础类型、自定义类型、切片等）只要实现了接口所要求的方法集，就自动被视为实现了该接口。

	// 打印排序后的结果
	fmt.Println(fruits)
}

// 问题：
// 为什么 Go 的 sort 包中接口的名字直接叫 Interface，而不是 SortInterface 或 IInterface 之类更易区分的名字？

// 回答：
// ✅ Go 的设计哲学之一是：在包的作用域内，名字应该尽量简洁。
// 在 sort 包内，Interface 这个名字已经足够明确表达“排序接口”的含义。
// 使用时通过包名前缀（sort.Interface）就能清晰地区分含义。

// 👉 这种设计风格强调通过“包名+接口名”的组合补充语义，而不是靠接口名自身堆砌前缀。

// 🆚 如果是 Java / C++ 风格：
// 常见命名方式可能是：
// - Sortable
// - SortInterface
// - ISortable

// ✅ Go 明确不推荐这种冗余命名。
// 原因：Go 提倡“简洁性优先”、“可读性优先”、“避免赘余”——这是语言的核心美学之一。

// ✅ 类似命名风格的例子还有：
// - sort.Interface：排序接口
// - io.Reader：表示“可读”的对象
// - io.Writer：表示“可写”的对象
// - http.Handler：HTTP 请求处理器

// ✅ 总结：
// Go 鼓励将“上下文”信息放在包名中，而不是在名字中重复描述。
// 所以接口名简洁、语义由“包名+类型名”共同组成。
// 虽然刚学 Go 时可能觉得不直观，但久了会体会到其清晰、自然、优雅。
