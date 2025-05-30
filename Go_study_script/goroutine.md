
# 🧵 Go 并发核心：Goroutine 全面总结（含练习题与答案）

本文结合 [draven.co Goroutine 原理章节](https://draven.co/golang/docs/part3-runtime/ch06-concurrency/golang-goroutine/) 及实际开发经验，系统整理 Goroutine 的核心原理与常见应用，并提供完整练习题与解答代码。

---

## ✅ 什么是 Goroutine？

- Go 语言内置的 **轻量级协程**
- 启动方式简单：`go func()`
- 自动调度，支持高并发，栈空间小（初始 2KB）
- 由 runtime 管理，不需要开发者手动管理线程

---

## 🧱 Goroutine 底层结构

每个 Goroutine 在 runtime 中是一个 `g` 结构体，包含：

- 栈信息
- 调度状态
- panic 链
- 当前执行状态等

---

## ⚙️ GMP 调度模型

| 组件 | 含义 |
|------|------|
| G（Goroutine） | 用户代码执行单元 |
| M（Machine） | 操作系统线程 |
| P（Processor） | 执行 G 的上下文，管理本地队列 |

---

## 🔁 Goroutine 创建与调度过程

1. 编译器将 `go f()` 转换为对 `runtime.newproc` 的调用
2. 创建新的 G 对象，挂入本地 P 的队列中
3. M 线程循环从 P 获取 G 执行

---

## 💤 状态转换

Goroutine 生命周期状态：

- `_Grunnable`：就绪态
- `_Grunning`：运行态
- `_Gwaiting`：阻塞态（等待 channel / sleep 等）
- `_Gdead`：结束态

---

## 🧠 调度时机

- 阻塞操作（channel、syscall）
- 调用 `runtime.Gosched()` 主动让出
- 被抢占
- 主动退出 `runtime.Goexit()`

---

## 🔧 栈空间优化

- 每个 Goroutine 默认 2KB 栈空间
- 遇到栈空间不足时自动增长
- 相比线程 1MB 更节省资源

---

## 📚 Goroutine 练习题与答案

---

### 🧪 练习 1：并发打印（使用 WaitGroup）

❓题目：使用多个 goroutine 并发打印 `"Hello from goroutine X"`（X 是序号），要求主程序等待所有 goroutine 结束后再退出。

✅ 解答：

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    var wg sync.WaitGroup

    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            fmt.Println("Hello from goroutine", i)
        }(i)
    }

    wg.Wait()
}
```

---

### 🧪 练习 2：channel + 多 worker 通信

❓题目：启动多个 goroutine，每个从独立的 channel 中接收一个整数并打印出来，要求打印格式为 "Worker X received: Y"。

✅ 解答：

```go
package main

import (
    "fmt"
)

func worker(id int, ch chan int) {
    num := <-ch
    fmt.Printf("Worker %d received: %d\n", id, num)
}

func main() {
    for i := 0; i < 3; i++ {
        ch := make(chan int)
        go worker(i, ch)
        ch <- i * 10
    }
}
```

---

### 🧪 练习 3：goroutine 泄漏修复

❓题目：以下 goroutine 永久阻塞导致资源泄漏，请设计合理的退出机制。

```go
func leaky() {
    ch := make(chan int)
    go func() {
        for {
            <-ch
        }
    }()
}
```

✅ 解答（加入 done 通道控制退出）：

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    ch := make(chan int)
    done := make(chan struct{})

    go func() {
        for {
            select {
            case <-done:
                fmt.Println("Exiting goroutine")
                return
            case v := <-ch:
                fmt.Println("Received:", v)
            }
        }
    }()

    ch <- 1
    time.Sleep(1 * time.Second)
    close(done)
}
```

---

### 🧪 练习 4：使用 context 控制 goroutine 生命周期

❓题目：创建一个 goroutine 执行任务，并在 2 秒后自动终止，要求使用 `context` 控制其生命周期。

✅ 解答：

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Worker canceled:", ctx.Err())
            return
        default:
            fmt.Println("Working...")
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    go worker(ctx)
    time.Sleep(3 * time.Second)
    fmt.Println("Main finished")
}
```

---

## ⚠️ 常见错误模式总结

| 问题类型 | 描述 | 正确做法 |
|----------|------|-----------|
| 泄漏 | 无退出机制 | 使用 context 或 done chan |
| 无限阻塞 | channel 无读端 | 使用带缓冲/设置超时 |
| 启动过多 | goroutine 无限制创建 | 建立 worker pool |
| 闭包捕获误用 | 循环变量 i 被所有 goroutine 共享 | 使用 `(i int)` 参数传递副本 |

---

## ✅ 总结一句话

> Goroutine 是 Go 并发的基石。学会它、管好它，配合 channel 和 context，才能真正写出安全、高效的并发程序。
