package main

//
//import "fmt"
//
//// 引入 fmt 包用于打印输出
//
//// 定义一个结构体类型 rect，表示一个矩形，包含两个字段：width 和 height
//type rect struct {
//	width, height int
//}
//
//// 定义一个方法 area，接收器为 *rect（指针类型），用于计算矩形面积
//func (r *rect) area() int {
//	return r.width * r.height // 通过指针访问字段，返回 width × height
//}
//
//// 定义一个方法 perim，接收器为 rect（值类型），用于计算矩形周长
//func (r rect) perim() int {
//	return 2*r.width + 2*r.height
//}
//
//func main() {
//	// 创建一个 rect 结构体实例 r，宽度为 10，高度为 5
//	r := rect{width: 10, height: 5}
//
//	// 调用 area 方法（接收器是 *rect）
//	// Go 自动将 r 转为 &r 调用，因为方法接收器是指针类型
//	fmt.Println("area: ", r.area()) // 输出：area:  50
//
//	// 调用 perim 方法（接收器是 rect 值类型）
//	fmt.Println("perim: ", r.perim()) // 输出：perim:  30
//
//	// 创建结构体指针 rp，指向 r
//	rp := &r
//
//	// 用指针调用指针接收器的方法，直接调用，无需自动转换
//	fmt.Println("area: ", rp.area()) // 输出：area:  50
//
//	// 用指针调用值接收器的方法，Go 自动解引用（rp → *rp）
//	fmt.Println("perim: ", rp.perim()) // 输出：perim:  30
//}
