
# Go 中各种类型 nil 的行为与比较规则详解

---

## ✅ nil 的适用类型

在 Go 中，以下类型可以被赋值为 nil：

- 指针：`*T`
- 接口：`interface{}`
- slice：`[]T`
- map：`map[K]V`
- channel：`chan T`
- function：`func(...)`
- unsafe.Pointer

---

## ✅ 不同类型 nil 的行为和对比

| 类型      | 零值是否为 nil | 是否可比较 | 是否可调用方法 | 是否能安全使用        |
|-----------|----------------|------------|----------------|------------------------|
| `*T`      | ✅ 是           | ✅ 可以     | ❌ panic        | ❌ 解引用会崩溃         |
| `[]T`     | ✅ 是           | ✅ 可以     | ✅ 部分方法     | ✅ append 可用         |
| `map[K]V` | ✅ 是           | ✅ 可以     | ✅ 只读可用     | ❌ 写入会 panic         |
| `chan T`  | ✅ 是           | ✅ 可以     | ❌ 收发阻塞     | ❌ 操作会 deadlock/panic |
| `func()`  | ✅ 是           | ✅ 可以     | ❌ panic        | ❌ 不能调用             |
| `interface{}` | ✅ 是       | ✅ 可以     | ✅ 但注意底层类型 | ❌ 复杂情况可能误判     |

---

## 🔍 接口 nil 判断陷阱

```go
var i interface{} = nil              // 完全 nil
var p *int = nil
var j interface{} = p               // j != nil !!!

fmt.Println(i == nil) // ✅ true
fmt.Println(j == nil) // ❌ false
```

解释：

- `i` 的类型和值都是 nil；
- `j` 的类型是 interface{}，底层值是 nil 指针，但 tab（类型信息）不为 nil；
- 因此 `j != nil`。

---

## ✅ 示例：slice 和 map 的 nil 行为对比

```go
var s []int         // nil slice
var m map[string]int // nil map

fmt.Println(len(s))     // 0
fmt.Println(s == nil)   // true
s = append(s, 1, 2)     // ✅ OK

fmt.Println(m == nil)   // true
_ = m["key"]            // ✅ OK
m["k"] = 1              // ❌ panic: assignment to entry in nil map
```

---

## ✅ 函数类型 nil 行为

```go
var f func() = nil

if f == nil {
    fmt.Println("f is nil")
}

f() // ❌ panic: call of nil function
```

---

## ✅ channel nil 行为

```go
var ch chan int

go func() { ch <- 1 }() // ❌ 永久阻塞
go func() { <-ch }()    // ❌ 永久阻塞
```

nil channel 是合法声明，但任何发送或接收都会永久阻塞！

---

## ✅ interface{} 和 nil 的组合图示（简化）

```text
var i interface{} = nil               => tab=nil, data=nil (i==nil)

var p *int = nil
var i interface{} = p                => tab=*int, data=nil (i!=nil)
```

---

## ✅ 总结口诀

```
nil 有类型，行为各异；
map 可读，写会炸；
slice append，无需怕；
接口判空需谨慎，tab 非空就不是 nil。
```

---

