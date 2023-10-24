# 功能概述
* 快如闪电的线程池
## 多线程示例
```go
func main() {
	pool := thread.NewClient(nil, 3)
	for i := 0; i < 20; i++ {
		_, err := pool.Write(&thread.Task{
			Func: func(ctx context.Context, i int) {
				log.Print(i, "start")
				time.Sleep(time.Second)
				log.Print(i, "end")
			},
			Args: []any{i},
		})
		if err != nil {
			log.Panic(err)
		}
	}
	pool.Join()
	log.Print("结束了")
}
```
## 获取线程id
```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/gospider007/thread"
)
func test(ctx context.Context, num int) {
	log.Printf("第%d个线程池中的第%d个请求开始", thread.GetThreadId(ctx), num)
	time.Sleep(time.Second)
	log.Printf("第%d个线程池中的第%d个请求结束", thread.GetThreadId(ctx), num)
}
func main() {
	threadCli := thread.NewClient(nil, 3) //限制并发为3
	for i := 0; i < 10; i++ {
		//读取任务
		threadCli.Write(&thread.Task{
			Func: test,
			Args: []any{i},
		})
	}
	threadCli.Join()
}
```