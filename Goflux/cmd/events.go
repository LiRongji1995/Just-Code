package cmd

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"

	"goflux/engine"
)

// handleJobEvents 是处理任务事件的核心函数
// 这里使用select循环同时监听多个channel
func handleJobEvents(ctx context.Context, job *engine.Job) error {
	var progressBar *pb.ProgressBar
	var lastProgress engine.ProgressUpdate

	// 初始化进度条（稍后会在收到第一个进度更新时设置）
	defer func() {
		if progressBar != nil {
			progressBar.Finish()
		}
	}()

	fmt.Printf("⏳ 等待任务开始...\n")

	// 核心事件循环
	for {
		select {
		// 监听进度更新
		case progress, ok := <-job.Progress():
			if !ok {
				// channel已关闭，任务结束
				return nil
			}

			// 更新进度显示
			if err := updateProgressDisplay(&progressBar, progress, &lastProgress); err != nil {
				fmt.Printf("⚠️  进度显示更新失败: %v\n", err)
			}

			lastProgress = progress

		// 监听错误事件
		case jobErr, ok := <-job.Errors():
			if !ok {
				// 错误channel已关闭
				continue
			}

			// 显示错误信息
			displayJobError(jobErr, progressBar != nil)

			if jobErr.Fatal {
				return fmt.Errorf("任务失败: %s", jobErr.Message)
			}

		// 监听任务完成信号
		case <-job.Done():
			// 任务完成，显示最终状态
			return handleJobCompletion(job, progressBar)

		// 监听外部取消信号
		case <-ctx.Done():
			fmt.Printf("\n🛑 任务被用户取消\n")
			job.Cancel()
			return nil
		}
	}
}

// updateProgressDisplay 更新进度条显示
func updateProgressDisplay(progressBar **pb.ProgressBar, progress engine.ProgressUpdate, lastProgress *engine.ProgressUpdate) error {
	// 状态变化时的特殊处理
	if lastProgress.Status != progress.Status {
		if *progressBar != nil {
			(*progressBar).Finish()
			fmt.Println() // 换行
		}

		// 显示状态变化
		displayStatusChange(progress)

		// 对于下载状态，初始化或重新初始化进度条
		if progress.Status == engine.JobStatusDownloading && progress.TotalSize > 0 {
			*progressBar = createProgressBar(progress)
		} else {
			*progressBar = nil // 清除进度条
		}
	}

	// 更新进度条
	if *progressBar != nil && progress.Status == engine.JobStatusDownloading {
		updateProgressBar(*progressBar, progress)
	}

	return nil
}

// displayStatusChange 显示状态变化
func displayStatusChange(progress engine.ProgressUpdate) {
	icon := getStatusIcon(progress.Status)
	statusName := getStatusDisplayName(progress.Status)

	fmt.Printf("📌 %s %s", icon, statusName)

	// 根据状态显示额外信息
	switch progress.Status {
	case engine.JobStatusMetadata:
		if progress.FileName != "" {
			fmt.Printf(": %s", progress.FileName)
		}
	case engine.JobStatusConnecting:
		if progress.TotalSize > 0 {
			fmt.Printf(" (%s)", engine.FormatBytes(progress.TotalSize))
		}
	case engine.JobStatusDownloading:
		if progress.FileName != "" && progress.TotalSize > 0 {
			fmt.Printf(": %s (%s)", progress.FileName, engine.FormatBytes(progress.TotalSize))
		}
	case engine.JobStatusSeeding:
		if progress.FileName != "" {
			fmt.Printf(": %s", progress.FileName)
		}
	}

	if progress.Message != "" {
		fmt.Printf(" - %s", progress.Message)
	}

	fmt.Println()
}

// createProgressBar 创建新的进度条
func createProgressBar(progress engine.ProgressUpdate) *pb.ProgressBar {
	bar := pb.Full.Start64(progress.TotalSize)
	bar.Set(pb.Bytes, true)

	// 自定义进度条模板
	tmpl := `{{with string . "prefix"}}{{.}} {{end}}{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{rtime . "ETA %s"}} {{with string . "suffix"}}{{.}}{{end}}`
	bar.SetTemplateString(tmpl)

	// 设置初始信息
	updateProgressBar(bar, progress)

	return bar
}

// updateProgressBar 更新进度条信息
func updateProgressBar(bar *pb.ProgressBar, progress engine.ProgressUpdate) {
	bar.SetCurrent(progress.DownloadedSize)

	// 构建前缀信息
	prefix := fmt.Sprintf("📥 %s", truncateFileName(progress.FileName, 20))
	if progress.ConnectedPeers > 0 {
		prefix += fmt.Sprintf(" [%d peers]", progress.ConnectedPeers)
	}
	bar.Set("prefix", prefix)

	// 构建后缀信息
	var suffixParts []string

	if progress.ActivePeers > 0 && progress.ActivePeers != progress.ConnectedPeers {
		suffixParts = append(suffixParts, fmt.Sprintf("active:%d", progress.ActivePeers))
	}

	if progress.TotalPieces > 0 {
		suffixParts = append(suffixParts, fmt.Sprintf("pieces:%d/%d",
			progress.CompletedPieces, progress.TotalPieces))
	}

	if len(suffixParts) > 0 {
		bar.Set("suffix", " ["+strings.Join(suffixParts, ", ")+"]")
	}
}

// displayJobError 显示任务错误
func displayJobError(jobErr *engine.JobError, hasProgressBar bool) {
	// 如果有进度条在运行，需要换行显示错误
	if hasProgressBar {
		fmt.Println()
	}

	var icon string
	if jobErr.Fatal {
		icon = "💥"
	} else {
		icon = "⚠️"
	}

	errorMsg := fmt.Sprintf("%s %s", icon, jobErr.Message)

	// 添加错误类型信息
	if jobErr.Type != "" {
		errorMsg += fmt.Sprintf(" [%s]", jobErr.Type)
	}

	// 添加错误代码
	if jobErr.Code != 0 {
		errorMsg += fmt.Sprintf(" (代码:%d)", jobErr.Code)
	}

	// 添加上下文信息
	if jobErr.Context != "" {
		errorMsg += fmt.Sprintf(" (%s)", jobErr.Context)
	}

	fmt.Println(errorMsg)
}

// handleJobCompletion 处理任务完成
func handleJobCompletion(job *engine.Job, progressBar *pb.ProgressBar) error {
	finalProgress := job.CurrentProgress()

	if progressBar != nil {
		progressBar.Finish()
		fmt.Println() // 换行
	}

	switch finalProgress.Status {
	case engine.JobStatusCompleted:
		return displayCompletionSuccess(finalProgress)
	case engine.JobStatusFailed:
		return displayCompletionFailure(finalProgress)
	default:
		return displayCompletionOther(finalProgress)
	}
}

// displayCompletionSuccess 显示成功完成
func displayCompletionSuccess(progress engine.ProgressUpdate) error {
	fmt.Printf("🎉 任务完成！\n")

	if progress.FileName != "" {
		fmt.Printf("📄 文件: %s\n", progress.FileName)
	}

	if progress.TotalSize > 0 {
		fmt.Printf("📏 大小: %s\n", engine.FormatBytes(progress.TotalSize))
	}

	if progress.ElapsedTime > 0 {
		fmt.Printf("⏱️  用时: %s\n", formatDuration(progress.ElapsedTime))

		// 计算平均速度
		if progress.TotalSize > 0 && progress.ElapsedTime.Seconds() > 0 {
			avgSpeed := float64(progress.TotalSize) / progress.ElapsedTime.Seconds()
			fmt.Printf("📊 平均速度: %s/s\n", engine.FormatBytes(int64(avgSpeed)))
		}
	}

	if progress.UploadedSize > 0 {
		fmt.Printf("📤 已上传: %s\n", engine.FormatBytes(progress.UploadedSize))

		// 计算分享率
		if progress.DownloadedSize > 0 {
			ratio := float64(progress.UploadedSize) / float64(progress.DownloadedSize)
			fmt.Printf("📈 分享率: %.2f\n", ratio)
		}
	}

	// 显示保存位置（如果是下载任务）
	if progress.Status == engine.JobStatusCompleted && progress.DownloadedSize > 0 {
		// 这里可以从job获取输出目录信息
		fmt.Printf("📁 保存位置: %s\n", getOutputPath(progress.FileName))
	}

	return nil
}

// displayCompletionFailure 显示失败完成
func displayCompletionFailure(progress engine.ProgressUpdate) error {
	fmt.Printf("❌ 任务失败: %s\n", progress.Message)

	if progress.DownloadedSize > 0 && progress.TotalSize > 0 {
		fmt.Printf("📊 已完成: %.1f%% (%s/%s)\n",
			progress.PercentComplete(),
			engine.FormatBytes(progress.DownloadedSize),
			engine.FormatBytes(progress.TotalSize))
	}

	if progress.ElapsedTime > 0 {
		fmt.Printf("⏱️  运行时间: %s\n", formatDuration(progress.ElapsedTime))
	}

	fmt.Println("\n💡 提示: 使用 --resume 选项可以尝试续传下载")

	return fmt.Errorf("任务执行失败")
}

// displayCompletionOther 显示其他完成状态
func displayCompletionOther(progress engine.ProgressUpdate) error {
	icon := getStatusIcon(progress.Status)
	statusName := getStatusDisplayName(progress.Status)

	fmt.Printf("%s 任务%s", icon, statusName)

	if progress.Message != "" {
		fmt.Printf(": %s", progress.Message)
	}

	fmt.Println()

	return nil
}

// 辅助函数：截断文件名
func truncateFileName(fileName string, maxLen int) string {
	if len(fileName) <= maxLen {
		return fileName
	}

	// 保留扩展名
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)

	if len(nameWithoutExt) <= maxLen-len(ext)-3 {
		return nameWithoutExt + "..." + ext
	}

	return nameWithoutExt[:maxLen-len(ext)-3] + "..." + ext
}

// 辅助函数：格式化持续时间
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), math.Mod(d.Seconds(), 60))
	} else {
		hours := int(d.Hours())
		minutes := int(math.Mod(d.Minutes(), 60))
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}

// 辅助函数：解析速度限制字符串 (如 "1MB/s", "500KB/s")
func parseSpeedLimit(limit string) (int64, error) {
	limit = strings.ToUpper(strings.TrimSpace(limit))
	if limit == "" || limit == "UNLIMITED" {
		return 0, nil
	}

	// 移除 "/S" 后缀
	limit = strings.TrimSuffix(limit, "/S")

	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(limit, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(limit, "KB")
	} else if strings.HasSuffix(limit, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(limit, "MB")
	} else if strings.HasSuffix(limit, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(limit, "GB")
	} else {
		numStr = limit
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析数字: %s", numStr)
	}

	return int64(num * float64(multiplier)), nil
}

// 辅助函数：解析时间持续时间
func parseDuration(duration string) (time.Duration, error) {
	return time.ParseDuration(duration)
}
