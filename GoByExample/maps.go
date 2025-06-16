package main

import "fmt"

func main() {
	m := make(map[string]int)

	m["k1"] = 7
	m["k2"] = 13

	fmt.Println("map:", m)

	v1 := m["k1"]
	fmt.Println("v1: ", v1)

	fmt.Println("len:", len(m))

	delete(m, "k2")
	fmt.Println("prs:", m)

	//Go 中访问 map 的语法可以返回两个值：value, exists := m["key"]
	//value 是键对应的值（若不存在，则为该值类型的零值）
	//exists 是一个 bool，表示该 key 是否真的存在于 map 中
	_, prs := m["k2"]
	fmt.Println("prs:", prs)

	n := map[string]int{"foo": 1, "bar": 2}
	fmt.Println("map:", n)
}
