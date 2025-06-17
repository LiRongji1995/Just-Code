package main

import "fmt" // 引入 fmt 包用于打印输出

// 定义一个结构体类型 person，包含两个字段：name 和 age
type person struct {
	name string
	age  int
}

// 定义一个构造函数 newPerson，接受 name 字符串，返回 *person 指针
func newPerson(name string) *person {
	p := person{name: name} // 创建一个 person 实例，只初始化 name 字段
	p.age = 42              // 手动设置 age 字段
	return &p               // 返回该结构体的指针（注意：返回局部变量的指针是安全的）
}

func main() {
	// 方式一：使用结构体字面量初始化所有字段
	fmt.Println(person{"Bob", 20}) // 输出：{Bob 20}

	// 方式二：命名字段初始化，顺序无关
	fmt.Println(person{name: "Alice", age: 30}) // 输出：{Alice 30}

	// 方式三：只初始化部分字段，其他字段取默认零值（int 为 0）
	fmt.Println(person{name: "Fred"}) // 输出：{Fred 0}

	// 方式四：使用结构体指针初始化（直接取地址）
	fmt.Println(&person{name: "Ann", age: 40}) // 输出：&{Ann 40}

	// 方式五：调用构造函数 newPerson 返回结构体指针
	fmt.Println(newPerson("Jon")) // 输出：&{Jon 42}

	// 创建结构体变量 s，并访问其字段
	s := person{name: "Sean", age: 50}
	fmt.Println(s.name) // 输出：Sean

	// 使用结构体指针 sp 指向 s
	sp := &s
	fmt.Println(sp.age) // 输出：50，Go 自动解引用（sp.age 等价于 (*sp).age）

	// 修改指针指向的结构体字段
	sp.age = 51
	fmt.Println(sp.age) // 输出：51，修改成功
}
