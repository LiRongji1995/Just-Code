// example.go - P2P引擎使用示例
package main

import (
	"context"
	"fmt"
	"goflux/engine"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	// 创建配置
	config := engine.DefaultConfig()

	// 自定义配置
	config.MaxPeers = 100
	config.SetSpeedLimits(10.0, 5.0) // 10MB/s下载，5MB/s上传
	config.AddTracker("http://tracker1.example.com:8080/announce")
	config.AddTracker("http://tracker2.example.com:8080/announce")
	config.WorkingDir = filepath.Join(os.Getenv("HOME"), "Downloads", "p2p")
	config.EnableDHT = true
	config.LogLevel = "info"

	fmt.Printf("使用配置:\n%s\n", config.String())

	// 创建引擎
	engine, err := engine.NewEngine(config)
	if err != nil {
		log.Fatalf("创建引擎失败: %v", err)
	}
	defer engine.Shutdown()

	// 运行内置测试（如果需要）
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		runEngineTests(engine)
		return
	}

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	setupGracefulShutdown(cancel, engine)

	// 示例1：创建下载任务
	if err := demonstrateDownload(ctx, engine); err != nil {
		log.Printf("下载示例失败: %v", err)
	}

	// 示例2：创建做种任务
	if err := demonstrateSeed(ctx, engine); err != nil {
		log.Printf("做种示例失败: %v", err)
	}

	// 示例3：监控引擎状态
	go monitorEngineStats(ctx, engine)

	// 等待信号
	<-ctx.Done()
	fmt.Println("程序正在退出...")
}

// 演示下载功能
func demonstrateDownload(ctx context.Context, eng *engine.Engine) error {
	fmt.Println("\n=== 下载示例 ===")

	// 创建下载任务
	job, err := eng.CreateDownloadJob("example.meta", "/tmp/downloads")
	if err != nil {
		return fmt.Errorf("创建下载任务失败: %w", err)
	}

	fmt.Printf("创建下载任务: %s\n", job.ID())

	// 监控任务进度
	go func() {
		for {
			select {
			case progress := <-job.Progress():
				fmt.Printf("下载进度: %.1f%% - %s\n",
					progress.PercentComplete(), progress.Message)

				if progress.IsComplete() {
					fmt.Printf("下载完成! 总大小: %s, 耗时: %s\n",
						progress.FormatSize(progress.TotalSize), progress.ElapsedTime)
					return
				}

			case err := <-job.Errors():
				if err.Fatal {
					fmt.Printf("致命错误: %s\n", err.Error())
					return
				} else {
					fmt.Printf("警告: %s\n", err.Error())
				}

			case <-job.Done():
				fmt.Printf("任务 %s 已完成\n", job.ID())
				return

			case <-ctx.Done():
				fmt.Println("下载任务被取消")
				job.Cancel()
				return
			}
		}
	}()

	// 模拟暂停和恢复
	time.Sleep(2 * time.Second)
	if err := job.Pause(); err == nil {
		fmt.Println("任务已暂停")
		time.Sleep(1 * time.Second)
		if err := job.Resume(); err == nil {
			fmt.Println("任务已恢复")
		}
	}

	return nil
}

// 演示做种功能
func demonstrateSeed(ctx context.Context, eng *engine.Engine) error {
	fmt.Println("\n=== 做种示例 ===")

	// 创建做种任务
	job, err := eng.CreateSeedJob("/path/to/local/file.dat", true)
	if err != nil {
		return fmt.Errorf("创建做种任务失败: %w", err)
	}

	fmt.Printf("创建做种任务: %s\n", job.ID())

	// 监控做种进度
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case progress := <-job.Progress():
				if progress.Status == engine.JobStatusSeeding {
					fmt.Printf("做种状态: 连接 %d 个Peer, 分享率: %.2f, 上传速度: %s\n",
						progress.ConnectedPeers,
						progress.Ratio,
						progress.FormatSpeed(progress.UploadSpeed))
				} else {
					fmt.Printf("做种准备: %s\n", progress.Message)
				}

			case err := <-job.Errors():
				fmt.Printf("做种错误: %s\n", err.Error())

			case <-job.Done():
				fmt.Printf("做种任务 %s 已结束\n", job.ID())
				return

			case <-ctx.Done():
				fmt.Println("做种任务被取消")
				job.Cancel()
				return

			case <-ticker.C:
				// 定期显示做种状态
				status := job.CurrentProgress()
				if status.Status == engine.JobStatusSeeding {
					fmt.Printf("💾 做种中... 上传: %s, 连接: %d\n",
						status.FormatSize(status.UploadedSize),
						status.ConnectedPeers)
				}
			}
		}
	}()

	return nil
}

// 监控引擎统计信息
func monitorEngineStats(ctx context.Context, eng *engine.Engine) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := eng.Stats()
			jobs := eng.ListJobs()

			fmt.Printf("\n📊 引擎统计 (运行时间: %s):\n", stats.Uptime.Round(time.Second))
			fmt.Printf("  活跃任务: %d\n", stats.ActiveJobs)
			fmt.Printf("  总下载: %s\n", engine.FormatBytes(stats.TotalDownloaded))
			fmt.Printf("  总上传: %s\n", engine.FormatBytes(stats.TotalUploaded))
			fmt.Printf("  连接数: %d\n", stats.ConnectedPeers)

			// 显示各个任务的状态
			for _, job := range jobs {
				progress := job.CurrentProgress()
				fmt.Printf("  任务 %s: %s (%.1f%%)\n",
					job.ID()[:8], progress.Status, progress.PercentComplete())
			}

		case <-ctx.Done():
			return
		}
	}
}

// 运行引擎测试
func runEngineTests(eng *engine.Engine) {
	fmt.Println("\n=== 运行引擎测试 ===")

	// 注意：这需要引擎在测试模式下初始化
	// 在实际使用中，您可能需要重新创建引擎并设置测试模式

	if err := eng.RunTests(); err != nil {
		fmt.Printf("测试失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 所有测试通过")
}

// 设置优雅关闭
func setupGracefulShutdown(cancel context.CancelFunc, eng *engine.Engine) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n收到信号: %v, 开始优雅关闭...\n", sig)

		// 取消上下文
		cancel()

		// 关闭引擎
		if err := eng.Shutdown(); err != nil {
			fmt.Printf("引擎关闭时出错: %v\n", err)
		}

		fmt.Println("引擎已安全关闭")
		os.Exit(0)
	}()
}

// 高级使用示例：批量任务管理
func demonstrateBatchOperations(ctx context.Context, eng *engine.Engine) error {
	fmt.Println("\n=== 批量任务管理示例 ===")

	// 批量创建下载任务
	metaFiles := []string{
		"movie1.meta",
		"software2.meta",
		"document3.meta",
	}

	var jobs []*engine.Job
	for i, metaFile := range metaFiles {
		outputDir := fmt.Sprintf("/tmp/downloads/batch_%d", i)
		job, err := eng.CreateDownloadJob(metaFile, outputDir)
		if err != nil {
			fmt.Printf("创建任务失败 %s: %v\n", metaFile, err)
			continue
		}
		jobs = append(jobs, job)
		fmt.Printf("创建批量任务 %d: %s\n", i+1, job.ID())
	}

	// 统一监控所有任务
	completed := 0
	for _, job := range jobs {
		go func(j *engine.Job) {
			for {
				select {
				case progress := <-j.Progress():
					if progress.IsComplete() {
						fmt.Printf("✅ 批量任务完成: %s\n", j.ID()[:8])
						completed++
						return
					}

				case err := <-j.Errors():
					if err.Fatal {
						fmt.Printf("❌ 批量任务失败: %s - %s\n", j.ID()[:8], err.Message)
						completed++ // 也算完成（虽然失败了）
						return
					}

				case <-j.Done():
					return

				case <-ctx.Done():
					j.Cancel()
					return
				}
			}
		}(job)
	}

	// 等待所有任务完成或超时
	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for completed < len(jobs) {
		select {
		case <-ticker.C:
			fmt.Printf("批量进度: %d/%d 任务完成\n", completed, len(jobs))

		case <-timeout.C:
			fmt.Printf("批量任务超时，取消剩余任务\n")
			for _, job := range jobs {
				if job.Status() != engine.JobStatusCompleted &&
					job.Status() != engine.JobStatusFailed {
					job.Cancel()
				}
			}
			return fmt.Errorf("batch operations timed out")

		case <-ctx.Done():
			fmt.Printf("批量任务被用户取消\n")
			return ctx.Err()
		}
	}

	fmt.Printf("🎉 所有批量任务完成! (%d/%d)\n", completed, len(jobs))
	return nil
}

// 演示任务队列和优先级管理
func demonstrateTaskQueue(ctx context.Context, eng *engine.Engine) error {
	fmt.Println("\n=== 任务队列管理示例 ===")

	// 创建不同优先级的任务
	highPriorityTasks := []string{"urgent.meta", "important.meta"}
	normalTasks := []string{"normal1.meta", "normal2.meta", "normal3.meta"}
	lowPriorityTasks := []string{"background1.meta", "background2.meta"}

	var allJobs []*engine.Job

	// 高优先级任务
	for _, meta := range highPriorityTasks {
		job, err := eng.CreateDownloadJob(meta, "/tmp/high_priority")
		if err != nil {
			continue
		}
		allJobs = append(allJobs, job)
		fmt.Printf("🔥 高优先级任务: %s\n", job.ID()[:8])
	}

	// 普通任务
	for _, meta := range normalTasks {
		job, err := eng.CreateDownloadJob(meta, "/tmp/normal")
		if err != nil {
			continue
		}
		allJobs = append(allJobs, job)
		fmt.Printf("📋 普通任务: %s\n", job.ID()[:8])
	}

	// 低优先级任务
	for _, meta := range lowPriorityTasks {
		job, err := eng.CreateDownloadJob(meta, "/tmp/low_priority")
		if err != nil {
			continue
		}
		allJobs = append(allJobs, job)
		fmt.Printf("⏳ 低优先级任务: %s\n", job.ID()[:8])
	}

	// 监控队列状态
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := eng.Stats()
				fmt.Printf("📊 队列状态: %d 个活跃任务\n", stats.ActiveJobs)

				// 显示每个任务的当前状态
				for i, job := range allJobs {
					if i >= 5 { // 只显示前5个避免刷屏
						break
					}
					progress := job.CurrentProgress()
					fmt.Printf("  [%d] %s: %s (%.1f%%)\n",
						i+1, job.ID()[:6], progress.Status, progress.PercentComplete())
				}
				fmt.Println()

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// 演示错误处理和恢复
func demonstrateErrorHandling(ctx context.Context, eng *engine.Engine) error {
	fmt.Println("\n=== 错误处理和恢复示例 ===")

	// 创建一个可能失败的任务
	job, err := eng.CreateDownloadJob("problematic.meta", "/tmp/test_errors")
	if err != nil {
		return fmt.Errorf("创建测试任务失败: %w", err)
	}

	fmt.Printf("创建测试任务: %s\n", job.ID())

	// 专门监听错误
	errorCount := 0
	go func() {
		for {
			select {
			case err := <-job.Errors():
				errorCount++
				fmt.Printf("🚨 错误 #%d [%s]: %s\n",
					errorCount, err.Type, err.Message)

				if err.Fatal {
					fmt.Printf("💀 致命错误，任务将停止\n")
					return
				} else {
					fmt.Printf("⚠️  非致命错误，任务继续\n")
				}

			case <-job.Done():
				fmt.Printf("📋 错误处理测试任务完成，共遇到 %d 个错误\n", errorCount)
				return

			case <-ctx.Done():
				job.Cancel()
				return
			}
		}
	}()

	// 模拟一些操作
	time.Sleep(1 * time.Second)

	// 展示任务控制
	fmt.Println("🎮 任务控制演示:")
	fmt.Println("  暂停任务...")
	if err := job.Pause(); err != nil {
		fmt.Printf("  暂停失败: %v\n", err)
	} else {
		fmt.Println("  ✅ 任务已暂停")

		time.Sleep(2 * time.Second)

		fmt.Println("  恢复任务...")
		if err := job.Resume(); err != nil {
			fmt.Printf("  恢复失败: %v\n", err)
		} else {
			fmt.Println("  ✅ 任务已恢复")
		}
	}

	return nil
}

// 演示性能监控和调优
func demonstratePerformanceMonitoring(ctx context.Context, eng *engine.Engine) {
	fmt.Println("\n=== 性能监控示例 ===")

	// 启动性能监控协程
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := eng.Stats()
				jobs := eng.ListJobs()

				// 计算总体性能指标
				var totalDownloadSpeed int64
				var totalUploadSpeed int64
				activeJobs := 0

				for _, job := range jobs {
					progress := job.CurrentProgress()
					if progress.IsActive() {
						totalDownloadSpeed += progress.DownloadSpeed
						totalUploadSpeed += progress.UploadSpeed
						activeJobs++
					}
				}

				fmt.Printf("🔍 性能监控报告:\n")
				fmt.Printf("  系统运行时间: %s\n", stats.Uptime.Round(time.Second))
				fmt.Printf("  活跃任务数: %d\n", activeJobs)
				fmt.Printf("  总下载速度: %s\n", engine.FormatBytesPerSecond(totalDownloadSpeed))
				fmt.Printf("  总上传速度: %s\n", engine.FormatBytesPerSecond(totalUploadSpeed))
				fmt.Printf("  总传输量: 下载 %s, 上传 %s\n",
					engine.FormatBytes(stats.TotalDownloaded),
					engine.FormatBytes(stats.TotalUploaded))
				fmt.Printf("  连接数: %d\n", stats.ConnectedPeers)

				// 性能建议
				if totalDownloadSpeed > 0 {
					efficiency := float64(stats.TotalDownloaded) / float64(stats.TotalDownloaded+stats.TotalUploaded) * 100
					fmt.Printf("  传输效率: %.1f%%\n", efficiency)

					if totalDownloadSpeed < 1024*1024 { // < 1MB/s
						fmt.Printf("  💡 建议: 考虑增加更多Peer连接以提高下载速度\n")
					}
				}

				fmt.Println()

			case <-ctx.Done():
				return
			}
		}
	}()
}

// 演示配置动态调整
func demonstrateDynamicConfiguration(eng *engine.Engine) {
	fmt.Println("\n=== 动态配置调整示例 ===")

	// 模拟根据网络状况调整配置
	fmt.Println("🔧 根据网络状况调整配置...")

	stats := eng.Stats()

	// 根据当前性能调整
	if stats.ConnectedPeers < 10 {
		fmt.Println("  检测到连接数较少，建议:")
		fmt.Println("    - 增加最大Peer连接数")
		fmt.Println("    - 启用DHT网络发现")
		fmt.Println("    - 添加更多Tracker服务器")
	}

	if stats.TotalDownloaded > 1024*1024*1024 { // > 1GB
		fmt.Println("  检测到大量下载，建议:")
		fmt.Println("    - 适当限制上传速度以优化下载")
		fmt.Println("    - 增加并发下载任务数")
		fmt.Println("    - 清理临时文件释放磁盘空间")
	}

	fmt.Println("  💡 提示: 可以通过配置文件或API动态调整这些参数")
}

// 主函数的完整版本，包含所有示例
func runCompleteDemo() {
	config := engine.DefaultConfig()
	config.MaxPeers = 50
	config.SetSpeedLimits(5.0, 2.0) // 5MB/s下载，2MB/s上传

	eng, err := engine.NewEngine(config)
	if err != nil {
		log.Fatalf("创建引擎失败: %v", err)
	}
	defer eng.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	setupGracefulShutdown(cancel, eng)

	// 运行所有演示
	fmt.Println("🚀 开始完整功能演示...")

	// 基础功能
	demonstrateDownload(ctx, eng)
	time.Sleep(2 * time.Second)

	demonstrateSeed(ctx, eng)
	time.Sleep(2 * time.Second)

	// 高级功能
	demonstrateBatchOperations(ctx, eng)
	time.Sleep(2 * time.Second)

	demonstrateTaskQueue(ctx, eng)
	time.Sleep(2 * time.Second)

	demonstrateErrorHandling(ctx, eng)
	time.Sleep(2 * time.Second)

	// 监控和性能
	demonstratePerformanceMonitoring(ctx, eng)
	time.Sleep(2 * time.Second)

	demonstrateDynamicConfiguration(eng)

	fmt.Println("\n🎉 演示完成! 引擎将继续运行直到收到停止信号...")
	<-ctx.Done()
}
