package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 5; i++ {
		go fmt.Println("Hello from goroutine", i)
	}
	time.Sleep(time.Second)
}
