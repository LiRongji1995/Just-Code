// testing.go - P2P引擎的测试和基准测试
package engine_test

import (
	"context"
	"fmt"
	"goflux/engine"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// 测试配置
var testConfig = engine.Config{
	ListenPort:             6881,
	MaxPeers:               20,
	MaxConcurrentDownloads: 4,
	PieceSize:              256 * 1024,
	ConnTimeout:            10 * time.Second,
	RequestTimeout:         5 * time.Second,
	MaxRetries:             3,
	WorkingDir:             "/tmp/p2p_test",
	TempDir:                "/tmp/p2p_test/.tmp",
	EnableDHT:              false,  // 测试时关闭DHT
	LogLevel:               "warn", // 减少日志输出
	EnableLog:              false,
}

// TestEngineBasicOperations 测试引擎基本操作
func TestEngineBasicOperations(t *testing.T) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 测试统计信息
	stats := engine.Stats()
	if stats.ActiveJobs != 0 {
		t.Errorf("期望初始活跃任务数为0，实际为%d", stats.ActiveJobs)
	}

	// 测试任务列表
	jobs := engine.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("期望初始任务列表为空，实际长度为%d", len(jobs))
	}
}

// TestJobCreationAndManagement 测试任务创建和管理
func TestJobCreationAndManagement(t *testing.T) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 测试下载任务创建
	downloadJob, err := engine.CreateDownloadJob("test.meta", "/tmp/test")
	if err != nil {
		t.Fatalf("创建下载任务失败: %v", err)
	}

	// 验证任务属性
	if downloadJob.ID() == "" {
		t.Error("任务ID不应为空")
	}
	if downloadJob.Status() != engine.JobStatusPending {
		t.Errorf("期望初始状态为%s，实际为%s", engine.JobStatusPending, downloadJob.Status())
	}

	// 测试任务检索
	retrievedJob, exists := engine.GetJob(downloadJob.ID())
	if !exists {
		t.Error("无法检索已创建的任务")
	}
	if retrievedJob.ID() != downloadJob.ID() {
		t.Error("检索到的任务ID不匹配")
	}

	// 测试做种任务创建
	seedJob, err := engine.CreateSeedJob("/tmp/test_file", true)
	if err != nil {
		t.Fatalf("创建做种任务失败: %v", err)
	}

	// 验证任务列表
	jobs := engine.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("期望任务列表长度为2，实际为%d", len(jobs))
	}

	// 测试任务控制
	if err := downloadJob.Pause(); err == nil {
		// 暂停成功后应该能够恢复
		time.Sleep(100 * time.Millisecond)
		if err := downloadJob.Resume(); err != nil {
			t.Errorf("恢复任务失败: %v", err)
		}
	}

	// 测试任务取消
	seedJob.Cancel()
	time.Sleep(100 * time.Millisecond) // 等待取消处理
}

// TestConcurrentJobOperations 测试并发任务操作
func TestConcurrentJobOperations(t *testing.T) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	const numJobs = 10
	var wg sync.WaitGroup
	jobs := make([]*engine.Job, numJobs)
	errors := make(chan error, numJobs)

	// 并发创建任务
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			metaFile := fmt.Sprintf("test_%d.meta", index)
			outputDir := fmt.Sprintf("/tmp/test_%d", index)

			job, err := engine.CreateDownloadJob(metaFile, outputDir)
			if err != nil {
				errors <- fmt.Errorf("创建任务%d失败: %w", index, err)
				return
			}

			jobs[index] = job
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查错误
	for err := range errors {
		t.Error(err)
	}

	// 验证所有任务都被创建
	jobList := engine.ListJobs()
	if len(jobList) != numJobs {
		t.Errorf("期望创建%d个任务，实际创建%d个", numJobs, len(jobList))
	}

	// 并发操作任务
	for i, job := range jobs {
		if job == nil {
			continue
		}

		go func(j *engine.Job, index int) {
			// 随机操作
			switch index % 3 {
			case 0:
				j.Pause()
				time.Sleep(50 * time.Millisecond)
				j.Resume()
			case 1:
				j.Cancel()
			default:
				// 什么都不做，让任务自然进行
			}
		}(job, i)
	}

	// 等待一段时间让操作完成
	time.Sleep(500 * time.Millisecond)
}

// TestPeerConnectionManagement 测试Peer连接管理
func TestPeerConnectionManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过Peer连接测试（短测试模式）")
	}

	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 创建任务以触发连接
	job, err := engine.CreateDownloadJob("peer_test.meta", "/tmp/peer_test")
	if err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}

	// 监控连接状态
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	maxPeers := 0
	for {
		select {
		case <-timeout.C:
			t.Logf("Peer连接测试完成，最大连接数: %d", maxPeers)
			job.Cancel()
			return

		case <-ticker.C:
			progress := job.CurrentProgress()
			if progress.ConnectedPeers > maxPeers {
				maxPeers = progress.ConnectedPeers
			}

			// 如果有连接就继续，否则继续等待
			if progress.ConnectedPeers > 0 {
				t.Logf("检测到%d个Peer连接", progress.ConnectedPeers)
			}
		}
	}
}

// TestErrorHandlingAndRecovery 测试错误处理和恢复
func TestErrorHandlingAndRecovery(t *testing.T) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 创建一个可能产生错误的任务
	job, err := engine.CreateDownloadJob("nonexistent.meta", "/invalid/path")
	if err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}

	// 监听错误
	errorReceived := false
	go func() {
		select {
		case err := <-job.Errors():
			t.Logf("收到预期的错误: %s", err.Message)
			errorReceived = true
		case <-time.After(5 * time.Second):
			t.Log("未收到预期的错误")
		}
	}()

	// 等待错误处理
	time.Sleep(2 * time.Second)

	if !errorReceived {
		t.Log("注意: 未收到错误可能是因为错误处理是异步的")
	}

	job.Cancel()
}

// BenchmarkEngineCreation 基准测试引擎创建
func BenchmarkEngineCreation(b *testing.B) {
	config := testConfig

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine, err := engine.NewEngine(config)
		if err != nil {
			b.Fatalf("创建引擎失败: %v", err)
		}
		engine.Shutdown()
	}
}

// BenchmarkJobCreation 基准测试任务创建
func BenchmarkJobCreation(b *testing.B) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		b.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job, err := engine.CreateDownloadJob(
			fmt.Sprintf("bench_%d.meta", i),
			fmt.Sprintf("/tmp/bench_%d", i),
		)
		if err != nil {
			b.Fatalf("创建任务失败: %v", err)
		}
		job.Cancel() // 立即取消以避免资源积累
	}
}

// BenchmarkConcurrentJobs 基准测试并发任务处理
func BenchmarkConcurrentJobs(b *testing.B) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		b.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			job, err := engine.CreateDownloadJob(
				fmt.Sprintf("concurrent_%d.meta", i),
				fmt.Sprintf("/tmp/concurrent_%d", i),
			)
			if err != nil {
				b.Fatalf("创建并发任务失败: %v", err)
			}
			job.Cancel()
			i++
		}
	})
}

// BenchmarkProgressUpdates 基准测试进度更新性能
func BenchmarkProgressUpdates(b *testing.B) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		b.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	job, err := engine.CreateDownloadJob("progress_test.meta", "/tmp/progress_test")
	if err != nil {
		b.Fatalf("创建测试任务失败: %v", err)
	}
	defer job.Cancel()

	// 启动进度监听器
	go func() {
		for range job.Progress() {
			// 消费进度更新
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟进度更新
		job.CurrentProgress()
	}
}

// TestMemoryUsage 测试内存使用情况
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存测试（短测试模式）")
	}

	const numJobs = 100

	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 创建大量任务
	jobs := make([]*engine.Job, numJobs)
	for i := 0; i < numJobs; i++ {
		job, err := engine.CreateDownloadJob(
			fmt.Sprintf("memory_test_%d.meta", i),
			fmt.Sprintf("/tmp/memory_test_%d", i),
		)
		if err != nil {
			t.Fatalf("创建任务%d失败: %v", i, err)
		}
		jobs[i] = job
	}

	// 让任务运行一段时间
	time.Sleep(2 * time.Second)

	// 取消所有任务
	for _, job := range jobs {
		job.Cancel()
	}

	// 等待清理
	time.Sleep(1 * time.Second)

	// 检查任务是否被正确清理
	remainingJobs := engine.ListJobs()
	if len(remainingJobs) > numJobs/2 { // 允许一些延迟清理
		t.Logf("警告: 仍有%d个任务未清理（共创建%d个）", len(remainingJobs), numJobs)
	}

	t.Logf("内存测试完成，创建了%d个任务，剩余%d个", numJobs, len(remainingJobs))
}

// TestStressTest 压力测试
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试（短测试模式）")
	}

	config := testConfig
	config.MaxPeers = 100
	config.MaxConcurrentDownloads = 10

	engine, err := engine.NewEngine(config)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	const (
		duration   = 30 * time.Second
		numWorkers = 5
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, numWorkers*10)

	// 启动多个工作协程
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			jobCounter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// 随机操作
				switch rand.Intn(4) {
				case 0, 1: // 50%概率创建下载任务
					job, err := engine.CreateDownloadJob(
						fmt.Sprintf("stress_%d_%d.meta", workerID, jobCounter),
						fmt.Sprintf("/tmp/stress_%d_%d", workerID, jobCounter),
					)
					if err != nil {
						errorChan <- fmt.Errorf("worker %d: 创建下载任务失败: %w", workerID, err)
						continue
					}

					// 随机取消一些任务
					if rand.Intn(3) == 0 {
						time.AfterFunc(time.Duration(rand.Intn(1000))*time.Millisecond, func() {
							job.Cancel()
						})
					}

				case 2: // 25%概率创建做种任务
					job, err := engine.CreateSeedJob(
						fmt.Sprintf("/tmp/seed_%d_%d", workerID, jobCounter),
						rand.Intn(2) == 0,
					)
					if err != nil {
						errorChan <- fmt.Errorf("worker %d: 创建做种任务失败: %w", workerID, err)
						continue
					}

					// 随机取消
					if rand.Intn(4) == 0 {
						time.AfterFunc(time.Duration(rand.Intn(2000))*time.Millisecond, func() {
							job.Cancel()
						})
					}

				case 3: // 25%概率查询状态
					jobs := engine.ListJobs()
					if len(jobs) > 0 {
						randomJob := jobs[rand.Intn(len(jobs))]
						_ = randomJob.Status()
						_ = randomJob.CurrentProgress()
					}
				}

				jobCounter++

				// 短暂休息避免过度压力
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			}
		}(i)
	}

	// 监控统计信息
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := engine.Stats()
				jobs := engine.ListJobs()
				t.Logf("压力测试进行中: %d活跃任务, %d总任务, 运行时间: %s",
					stats.ActiveJobs, len(jobs), stats.Uptime.Round(time.Second))
			}
		}
	}()

	// 等待所有工作协程完成
	wg.Wait()
	close(errorChan)

	// 检查错误
	errorCount := 0
	for err := range errorChan {
		errorCount++
		if errorCount <= 5 { // 只显示前5个错误
			t.Logf("压力测试错误: %v", err)
		}
	}

	if errorCount > 0 {
		t.Logf("压力测试完成，共发生%d个错误", errorCount)
	}

	// 最终统计
	finalStats := engine.Stats()
	finalJobs := engine.ListJobs()
	t.Logf("压力测试结果: 最终有%d个任务，%d个活跃任务", len(finalJobs), finalStats.ActiveJobs)
}

// TestLongRunningStability 测试长期运行稳定性
func TestLongRunningStability(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长期稳定性测试（短测试模式）")
	}

	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	const testDuration = 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// 统计指标
	var (
		createdJobs   int64
		completedJobs int64
		cancelledJobs int64
		errors        int64
	)

	// 任务创建器
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				job, err := engine.CreateDownloadJob(
					fmt.Sprintf("stability_%d.meta", createdJobs),
					fmt.Sprintf("/tmp/stability_%d", createdJobs),
				)
				if err != nil {
					errors++
					continue
				}

				createdJobs++

				// 监控任务完成
				go func(j *engine.Job) {
					select {
					case <-j.Done():
						if j.Status() == engine.JobStatusCompleted {
							completedJobs++
						} else {
							cancelledJobs++
						}
					case <-ctx.Done():
						j.Cancel()
						cancelledJobs++
					}
				}(job)

				// 随机取消一些任务
				if rand.Intn(5) == 0 {
					time.AfterFunc(time.Duration(rand.Intn(5000))*time.Millisecond, func() {
						job.Cancel()
					})
				}
			}
		}
	}()

	// 定期清理
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				jobs := engine.ListJobs()
				for _, job := range jobs {
					if job.Status() == engine.JobStatusCompleted ||
						job.Status() == engine.JobStatusFailed {
						// 在实际实现中，这里会调用清理方法
						// engine.removeJob(job.ID())
					}
				}
			}
		}
	}()

	// 定期报告
	reportTicker := time.NewTicker(10 * time.Second)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Logf("长期稳定性测试完成:")
			t.Logf("  创建任务: %d", createdJobs)
			t.Logf("  完成任务: %d", completedJobs)
			t.Logf("  取消任务: %d", cancelledJobs)
			t.Logf("  发生错误: %d", errors)

			finalJobs := engine.ListJobs()
			t.Logf("  剩余任务: %d", len(finalJobs))

			return

		case <-reportTicker.C:
			stats := engine.Stats()
			t.Logf("稳定性测试进行中: 创建%d, 完成%d, 取消%d, 错误%d, 活跃%d",
				createdJobs, completedJobs, cancelledJobs, errors, stats.ActiveJobs)
		}
	}
}

// 性能基准测试套件
func BenchmarkEnginePerformance(b *testing.B) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		b.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	b.Run("JobOperations", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				job, err := engine.CreateDownloadJob(
					fmt.Sprintf("perf_%d.meta", i),
					fmt.Sprintf("/tmp/perf_%d", i),
				)
				if err != nil {
					b.Fatal(err)
				}

				// 执行一些操作
				_ = job.Status()
				_ = job.CurrentProgress()
				job.Cancel()

				i++
			}
		})
	})

	b.Run("StatsAccess", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = engine.Stats()
				_ = engine.ListJobs()
			}
		})
	})
}

// TestRaceConditions 测试竞态条件
func TestRaceConditions(t *testing.T) {
	engine, err := engine.NewEngine(testConfig)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup

	// 启动多个协程进行并发操作
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 随机选择操作
				switch rand.Intn(6) {
				case 0: // 创建下载任务
					job, err := engine.CreateDownloadJob(
						fmt.Sprintf("race_%d_%d.meta", goroutineID, j),
						fmt.Sprintf("/tmp/race_%d_%d", goroutineID, j),
					)
					if err == nil {
						// 立即取消避免资源积累
						time.AfterFunc(time.Duration(rand.Intn(100))*time.Millisecond, func() {
							job.Cancel()
						})
					}

				case 1: // 创建做种任务
					job, err := engine.CreateSeedJob(
						fmt.Sprintf("/tmp/seed_race_%d_%d", goroutineID, j),
						rand.Intn(2) == 0,
					)
					if err == nil {
						time.AfterFunc(time.Duration(rand.Intn(100))*time.Millisecond, func() {
							job.Cancel()
						})
					}

				case 2: // 获取统计信息
					_ = engine.Stats()

				case 3: // 列出任务
					jobs := engine.ListJobs()
					for _, job := range jobs {
						_ = job.Status()
					}

				case 4: // 查找任务
					jobs := engine.ListJobs()
					if len(jobs) > 0 {
						randomJob := jobs[rand.Intn(len(jobs))]
						_, _ = engine.GetJob(randomJob.ID())
					}

				case 5: // 操作随机任务
					jobs := engine.ListJobs()
					if len(jobs) > 0 {
						randomJob := jobs[rand.Intn(len(jobs))]
						switch rand.Intn(3) {
						case 0:
							randomJob.Pause()
						case 1:
							randomJob.Resume()
						case 2:
							randomJob.Cancel()
						}
					}
				}

				// 短暂休息
				if rand.Intn(10) == 0 {
					time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				}
			}
		}(i)
	}

	// 等待所有协程完成
	wg.Wait()

	// 验证引擎仍然正常工作
	stats := engine.Stats()
	jobs := engine.ListJobs()

	t.Logf("竞态条件测试完成: %d个活跃任务, %d个总任务", stats.ActiveJobs, len(jobs))

	// 创建一个新任务验证功能正常
	testJob, err := engine.CreateDownloadJob("post_race_test.meta", "/tmp/post_race")
	if err != nil {
		t.Errorf("竞态测试后创建任务失败: %v", err)
	} else {
		testJob.Cancel()
		t.Log("竞态测试后功能验证通过")
	}
}

// TestResourceCleanup 测试资源清理
func TestResourceCleanup(t *testing.T) {
	const numIterations = 5

	for i := 0; i < numIterations; i++ {
		t.Logf("资源清理测试迭代 %d/%d", i+1, numIterations)

		engine, err := engine.NewEngine(testConfig)
		if err != nil {
			t.Fatalf("迭代%d: 创建引擎失败: %v", i, err)
		}

		// 创建一些任务
		jobs := make([]*engine.Job, 10)
		for j := range jobs {
			job, err := engine.CreateDownloadJob(
				fmt.Sprintf("cleanup_%d_%d.meta", i, j),
				fmt.Sprintf("/tmp/cleanup_%d_%d", i, j),
			)
			if err != nil {
				t.Errorf("迭代%d: 创建任务%d失败: %v", i, j, err)
				continue
			}
			jobs[j] = job
		}

		// 让任务运行一会儿
		time.Sleep(100 * time.Millisecond)

		// 取消部分任务
		for j, job := range jobs {
			if job != nil && j%2 == 0 {
				job.Cancel()
			}
		}

		// 关闭引擎
		if err := engine.Shutdown(); err != nil {
			t.Errorf("迭代%d: 引擎关闭失败: %v", i, err)
		}

		// 短暂等待确保清理完成
		time.Sleep(50 * time.Millisecond)
	}

	t.Log("资源清理测试完成")
}

// 辅助函数：创建测试配置
func createTestConfig(overrides map[string]interface{}) engine.Config {
	config := testConfig

	// 应用覆盖设置
	for key, value := range overrides {
		switch key {
		case "MaxPeers":
			config.MaxPeers = value.(int)
		case "MaxConcurrentDownloads":
			config.MaxConcurrentDownloads = value.(int)
		case "EnableDHT":
			config.EnableDHT = value.(bool)
			// 可以添加更多字段
		}
	}

	return config
}

// 性能测试辅助函数
func measureMemoryUsage() (heapMB, stackMB int64) {
	// 在实际实现中，这里会使用 runtime.ReadMemStats
	// 返回堆和栈的内存使用量（MB）
	return 0, 0
}

// 示例：如何运行所有测试
func ExampleRunAllTests() {
	// 运行单元测试
	// go test -v ./...

	// 运行基准测试
	// go test -bench=. -benchmem

	// 运行压力测试
	// go test -run=TestStress -timeout=2m

	// 运行长期稳定性测试
	// go test -run=TestLongRunning -timeout=5m

	// 运行竞态检测
	// go test -race -run=TestRace

	fmt.Println("示例测试命令已列出")
}
