package main

import "fmt" // 导入 fmt 包用于打印输出

// ping 函数：将 msg 写入到 pings 通道
// 注意：pings 的方向是 chan<- string，只能“发送”，不能接收
func ping(pings chan<- string, msg string) {
	pings <- msg // 将消息发送到 pings 通道中
}

// pong 函数：从 pings 通道接收数据，再发送到 pongs 通道
// pings 是 <-chan string，只能接收
// pongs 是 chan<- string，只能发送
func pong(pings <-chan string, pongs chan<- string) {
	msg := <-pings // 从 pings 通道中接收消息
	pongs <- msg   // 将该消息发送到 pongs 通道中
}

func main() {
	pings := make(chan string, 1) // 创建一个字符串通道 pings，带缓冲区容量为 1
	pongs := make(chan string, 1) // 创建一个字符串通道 pongs，带缓冲区容量为 1

	ping(pings, "passed message") // 向 pings 通道发送字符串消息
	pong(pings, pongs)            // 从 pings 通道接收消息并转发到 pongs 通道

	fmt.Println(<-pongs) // 从 pongs 通道中接收消息并打印，输出结果为：passed message
}
