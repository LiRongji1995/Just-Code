package main

import "fmt"

// 引入 fmt 包用于打印输出

// MapKeys 定义一个泛型函数 MapKeys，接受一个键为 K，值为 V 的 map，返回所有键组成的切片
// K 必须是可比较的（comparable），V 为任意类型（any）
func MapKeys[K comparable, V any](m map[K]V) []K {
	r := make([]K, 0, len(m)) // 创建一个初始容量为 map 长度的 K 类型切片
	for k := range m {        // 遍历 map 中的所有 key
		r = append(r, k) // 将 key 追加到结果切片中
	}
	return r // 返回包含所有 key 的切片
}

// 定义链表的节点结构 element[T]，包含一个值 val 和一个指向下一个节点的指针 next
type element[T any] struct {
	next *element[T] // 指向下一个节点
	val  T           // 存储的数据
}

// List 定义一个泛型链表结构 List，存储任意类型 T 的元素
type List[T any] struct {
	head, tail *element[T] // 链表头尾指针，指向 element[T]
}

// GetAll List[T] 的方法：GetAll，用于获取链表中的所有值，返回 []T 类型切片
func (lst *List[T]) GetAll() []T {
	var elems []T                             // 用于存储结果的切片
	for e := lst.head; e != nil; e = e.next { // 遍历链表，直到 e 为空
		elems = append(elems, e.val) // 将每个节点的值加入结果
	}
	return elems // 返回所有值组成的切片
}

// Push List[T] 的方法：Push，用于向链表末尾添加新元素 v（类型为 T）
func (lst *List[T]) Push(v T) {
	if lst.tail == nil { // 链表为空，插入第一个元素
		lst.head = &element[T]{val: v} // 创建新节点并设为 head
		lst.tail = lst.head            // tail 也指向 head
	} else { // 链表非空，在 tail 后追加新节点
		lst.tail.next = &element[T]{val: v} // 创建新节点并链接
		lst.tail = lst.tail.next            // 更新 tail 指针
	}
}

func main() {
	// 定义一个 map[int]string 类型的变量 m
	var m = map[int]string{1: "2", 2: "4", 3: "6"}

	// 打印 map 的所有 key，调用 MapKeys 自动类型推断
	fmt.Println("keys m :", MapKeys(m)) // 输出：keys m : [1 2 3]（顺序不保证）

	// 显式指定类型参数调用 MapKeys 函数
	_ = MapKeys[int, string](m) // 结果被丢弃

	// 创建一个 List[int] 泛型链表 lst
	lst := List[int]{}
	lst.Push(10) // 添加元素 10
	lst.Push(13) // 添加元素 13
	lst.Push(23) // 添加元素 23

	// 打印链表中所有元素
	fmt.Println("list:", lst.GetAll()) // 输出：list: [10 13 23]
}
