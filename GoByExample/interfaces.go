package main

import (
	"fmt"
	"math" // 导入 math 包，用于计算 π 和浮点数运算
)

// 定义一个接口类型 geometry，它包含两个方法：area() 和 perim()
type geometry interface {
	area() float64  // 计算面积
	perim() float64 // 计算周长
}

/*
✅使用 “Generate” 快捷键实现结构体接口方法

【快捷键】
- Windows: Alt + Insert
【步骤】
1. 将光标放在你定义的结构体体名附近（例如：type rect struct { ... }）。
2. 按下快捷键（Windows: Alt + Insert；macOS: Cmd + N）。
3. 在弹出的菜单中选择：Implement Methods...（实现方法）。
4. 在弹窗列表中选中你想实现的接口（如 geometry）。
5. 回车确认，GoLand 会自动生成该接口中未实现的方法签名！

💡 提示：生成的方法包含 `TODO` 注释，便于你后续填写具体实现逻辑。
*/

// 定义结构体 rect（矩形），包含宽度和高度
type rect struct {
	width, height float64
}

// 定义结构体 circle（圆），包含半径
type circle struct {
	radius float64
}

// rect 实现 geometry 接口的 area 方法
func (r rect) area() float64 {
	return r.width * r.height // 面积 = 宽 × 高
}

// rect 实现 geometry 接口的 perim 方法
func (r rect) perim() float64 {
	return 2*r.width + 2*r.height // 周长 = 2×宽 + 2×高
}

// circle 实现 geometry 接口的 area 方法
func (c circle) area() float64 {
	return math.Pi * c.radius * c.radius // 面积 = πr²
}

// circle 实现 geometry 接口的 perim 方法
func (c circle) perim() float64 {
	return 2 * math.Pi * c.radius // 周长 = 2πr
}

// 接受接口类型 geometry 的参数 g，并调用其方法
func measure(g geometry) {
	fmt.Println(g)         // 打印结构体本身（rect 或 circle）
	fmt.Println(g.area())  // 多态调用对应类型的 area 方法
	fmt.Println(g.perim()) // 多态调用对应类型的 perim 方法
}

func main() {
	r := rect{width: 3, height: 4} // 创建一个矩形实例
	c := circle{radius: 5}         // 创建一个圆实例

	measure(r) // 传入 rect 类型（自动识别为实现了 geometry 接口）
	measure(c) // 传入 circle 类型（同样实现了 geometry 接口）
}
