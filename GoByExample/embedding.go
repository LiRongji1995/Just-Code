package main

import "fmt" // 引入 fmt 包用于格式化输出

// 定义一个结构体 base，包含一个字段 num
type base struct {
	num int
}

// 为 base 类型定义一个方法 describe，返回一个格式化字符串
func (b base) describe() string {
	return fmt.Sprintf("base with num=%v", b.num)
}

// 定义另一个结构体 container，嵌入 base（匿名字段），并添加一个字符串字段 str
type container struct {
	base        // 匿名嵌入 base，base 的字段和方法会被“提升”为 container 的一部分
	str  string // container 自己的字段
}

func main() {
	// 创建一个 container 实例 co，初始化 base 和 str 两部分
	co := container{
		base: base{
			num: 1, // 初始化 base 中的 num 字段
		},
		str: "some name", // 初始化 container 自己的 str 字段
	}

	// 访问提升字段 num（来自 base），无需通过 co.base 显式访问
	fmt.Printf("co={num: %v, str: %v}\n", co.num, co.str)
	// 输出：co={num: 1, str: some name}

	// 显式通过 base 访问 num 字段
	fmt.Println("also num:", co.base.num)
	// 输出：also num: 1

	// 调用提升的方法 describe（来自 base），Go 会自动将调用委托给 co.base.describe()
	fmt.Println("describe:", co.describe())
	// 输出：describe: base with num=1

	// 定义一个接口 describer，要求实现一个方法 describe() string
	type describer interface {
		describe() string
	}

	// 声明一个 describer 接口变量 d，并赋值为 co
	var d describer = co
	// 由于 container 嵌入了 base，且 base 实现了 describe() 方法，container 也算实现了接口

	// 接口调用方法 describe，运行时会调 co.describe()，实际调用的是 base.describe()
	fmt.Println("describer:", d.describe())
	// 输出：describer: base with num=1
}
