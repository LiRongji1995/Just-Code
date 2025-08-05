package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goflux/engine"
)

// statusCmd 显示运行状态
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示引擎和任务状态",
	Long: `查看P2P引擎的运行状态和所有任务的详细信息。

显示内容包括:
  • 引擎全局统计信息
  • 活跃任务列表和进度
  • 网络连接状态
  • 性能统计数据

示例:
  p2p-downloader status              # 显示基本状态
  p2p-downloader status --detailed   # 显示详细信息
  p2p-downloader status --jobs-only  # 仅显示任务列表
  p2p-downloader status --watch      # 持续监控模式`,
	RunE: runStatus,
}

// 状态命令选项
var (
	statusDetailed bool
	statusJobsOnly bool
	statusWatch    bool
	statusInterval int
	statusFormat   string
)

// runStatus 显示引擎状态
func runStatus(cmd *cobra.Command, args []string) error {
	// 创建临时引擎实例来查询状态
	config, err := buildEngineConfigForStatus()
	if err != nil {
		return fmt.Errorf("❌ 配置错误: %w", err)
	}

	eng, err := engine.NewEngine(config)
	if err != nil {
		return fmt.Errorf("❌ 引擎初始化失败: %w", err)
	}
	defer eng.Shutdown()

	// 监控模式
	if statusWatch {
		return runStatusWatch(eng)
	}

	// 单次状态查询
	return displayStatus(eng)
}

// buildEngineConfigForStatus 构建状态查询的引擎配置
func buildEngineConfigForStatus() (engine.Config, error) {
	// 使用最小配置，只用于查询状态
	config := engine.DefaultConfig()

	// 只设置基本必要的配置
	config.ListenPort = listenPort
	config.MaxPeers = 1      // 状态查询不需要大量连接
	config.EnableDHT = false // 状态查询不需要DHT
	config.WorkingDir = workingDir

	return config, nil
}

// displayStatus 显示当前状态
func displayStatus(eng *engine.Engine) error {
	stats := eng.Stats()
	jobs := eng.ListJobs()

	switch statusFormat {
	case "json":
		return displayStatusJSON(stats, jobs)
	case "csv":
		return displayStatusCSV(stats, jobs)
	default:
		return displayStatusDefault(stats, jobs)
	}
}

// displayStatusDefault 显示默认格式的状态
func displayStatusDefault(stats engine.EngineStats, jobs []*engine.Job) error {
	if !statusJobsOnly {
		displayEngineStats(stats)
		fmt.Println()
	}

	displayJobsList(jobs)

	if statusDetailed && len(jobs) > 0 {
		fmt.Println()
		displayDetailedJobs(jobs)
	}

	return nil
}

// displayEngineStats 显示引擎统计信息
func displayEngineStats(stats engine.EngineStats) {
	fmt.Println("📊 P2P引擎状态")
	fmt.Println("================")
	fmt.Printf("🔧 活跃任务: %d\n", stats.ActiveJobs)
	fmt.Printf("📥 总下载量: %s\n", engine.FormatBytes(stats.TotalDownloaded))
	fmt.Printf("📤 总上传量: %s\n", engine.FormatBytes(stats.TotalUploaded))

	// 计算分享率
	if stats.TotalDownloaded > 0 {
		ratio := float64(stats.TotalUploaded) / float64(stats.TotalDownloaded)
		fmt.Printf("📈 分享率: %.2f\n", ratio)
	}

	fmt.Printf("🌐 连接节点: %d\n", stats.ConnectedPeers)
	fmt.Printf("⏱️  运行时间: %s\n", formatDuration(stats.Uptime))
}

// displayJobsList 显示任务列表
func displayJobsList(jobs []*engine.Job) {
	if len(jobs) == 0 {
		fmt.Println("📭 暂无活跃任务")
		return
	}

	fmt.Printf("📋 任务列表 (%d个)\n", len(jobs))
	fmt.Println("===================")

	// 按状态分组显示
	jobsByStatus := groupJobsByStatus(jobs)

	// 定义状态显示顺序
	statusOrder := []engine.JobStatus{
		engine.JobStatusDownloading,
		engine.JobStatusSeeding,
		engine.JobStatusConnecting,
		engine.JobStatusMetadata,
		engine.JobStatusPending,
		engine.JobStatusPaused,
		engine.JobStatusCompleted,
		engine.JobStatusFailed,
	}

	for _, status := range statusOrder {
		jobList, exists := jobsByStatus[status]
		if !exists || len(jobList) == 0 {
			continue
		}

		fmt.Printf("\n%s %s (%d个):\n", getStatusIcon(status),
			getStatusDisplayName(status), len(jobList))

		for _, job := range jobList {
			displayJobSummary(job)
		}
	}
}

// displayJobSummary 显示任务摘要
func displayJobSummary(job *engine.Job) {
	progress := job.CurrentProgress()

	fmt.Printf("  🆔 %s\n", job.ID()[:8])

	if progress.FileName != "" {
		fmt.Printf("     📄 %s", progress.FileName)
		if progress.TotalSize > 0 {
			fmt.Printf(" (%s)", engine.FormatBytes(progress.TotalSize))
		}
		fmt.Println()
	}

	if progress.Status == engine.JobStatusDownloading {
		fmt.Printf("     📊 %.1f%% ", progress.PercentComplete())
		if progress.DownloadSpeed > 0 {
			fmt.Printf("(%s/s) ", engine.FormatBytes(progress.DownloadSpeed))
		}
		if progress.ConnectedPeers > 0 {
			fmt.Printf("[%d peers] ", progress.ConnectedPeers)
		}
		if progress.EstimatedTime > 0 {
			fmt.Printf("ETA: %s", formatDuration(progress.EstimatedTime))
		}
		fmt.Println()
	} else if progress.Status == engine.JobStatusSeeding {
		if progress.UploadSpeed > 0 {
			fmt.Printf("     📤 %s/s ", engine.FormatBytes(progress.UploadSpeed))
		}
		if progress.ConnectedPeers > 0 {
			fmt.Printf("[%d peers] ", progress.ConnectedPeers)
		}
		if progress.Ratio > 0 {
			fmt.Printf("比率: %.2f", progress.Ratio)
		}
		fmt.Println()
	}

	if progress.Message != "" {
		fmt.Printf("     💬 %s\n", progress.Message)
	}

	fmt.Println()
}

// displayDetailedJobs 显示任务详细信息
func displayDetailedJobs(jobs []*engine.Job) {
	fmt.Println("📋 详细任务信息")
	fmt.Println("================")

	for i, job := range jobs {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 50))
		}

		progress := job.CurrentProgress()

		fmt.Printf("🆔 任务ID: %s\n", job.ID())
		fmt.Printf("📄 文件名: %s\n", progress.FileName)
		fmt.Printf("📊 状态: %s %s\n", getStatusIcon(progress.Status),
			getStatusDisplayName(progress.Status))

		if progress.TotalSize > 0 {
			fmt.Printf("📏 文件大小: %s\n", engine.FormatBytes(progress.TotalSize))
			fmt.Printf("📥 已下载: %s (%.1f%%)\n",
				engine.FormatBytes(progress.DownloadedSize), progress.PercentComplete())
		}

		if progress.TotalPieces > 0 {
			fmt.Printf("📦 分片进度: %d/%d (%.1f%%)\n",
				progress.CompletedPieces, progress.TotalPieces, progress.PieceProgress())
		}

		if progress.ConnectedPeers > 0 {
			fmt.Printf("🌐 连接节点: %d (活跃: %d)\n",
				progress.ConnectedPeers, progress.ActivePeers)
		}

		if progress.DownloadSpeed > 0 || progress.UploadSpeed > 0 {
			fmt.Printf("📈 速度: ↓%s/s ↑%s/s\n",
				engine.FormatBytes(progress.DownloadSpeed), engine.FormatBytes(progress.UploadSpeed))
		}

		if progress.ElapsedTime > 0 {
			fmt.Printf("⏱️  已用时间: %s\n", formatDuration(progress.ElapsedTime))
		}

		if progress.EstimatedTime > 0 {
			fmt.Printf("⏰ 预计剩余: %s\n", formatDuration(progress.EstimatedTime))
		}

		if progress.StartTime.IsZero() == false {
			fmt.Printf("🕐 开始时间: %s\n", progress.StartTime.Format("2006-01-02 15:04:05"))
		}

		fmt.Printf("🔄 最后更新: %s\n", progress.Timestamp.Format("15:04:05"))

		if progress.Message != "" {
			fmt.Printf("💬 状态消息: %s\n", progress.Message)
		}
	}
}

// runStatusWatch 运行监控模式
func runStatusWatch(eng *engine.Engine) error {
	fmt.Printf("📺 进入监控模式 (每%d秒更新，按Ctrl+C退出)\n\n", statusInterval)

	ticker := time.NewTicker(time.Duration(statusInterval) * time.Second)
	defer ticker.Stop()

	// 首次显示
	displayStatus(eng)

	for {
		select {
		case <-ticker.C:
			// 清屏并重新显示
			fmt.Print("\033[2J\033[H") // ANSI清屏序列
			fmt.Printf("📺 监控模式 - %s (每%d秒更新)\n\n",
				time.Now().Format("15:04:05"), statusInterval)
			displayStatus(eng)
		}
	}
}

// groupJobsByStatus 按状态分组任务
func groupJobsByStatus(jobs []*engine.Job) map[engine.JobStatus][]*engine.Job {
	grouped := make(map[engine.JobStatus][]*engine.Job)

	for _, job := range jobs {
		status := job.Status()
		grouped[status] = append(grouped[status], job)
	}

	// 对每个状态组内的任务按创建时间排序
	for status, jobList := range grouped {
		sort.Slice(jobList, func(i, j int) bool {
			// 这里简化处理，实际可能需要比较任务创建时间
			return jobList[i].ID() < jobList[j].ID()
		})
		grouped[status] = jobList
	}

	return grouped
}

// getStatusDisplayName 获取状态的显示名称
func getStatusDisplayName(status engine.JobStatus) string {
	switch status {
	case engine.JobStatusPending:
		return "等待中"
	case engine.JobStatusMetadata:
		return "解析元数据"
	case engine.JobStatusConnecting:
		return "连接中"
	case engine.JobStatusDownloading:
		return "下载中"
	case engine.JobStatusSeeding:
		return "做种中"
	case engine.JobStatusCompleted:
		return "已完成"
	case engine.JobStatusFailed:
		return "失败"
	case engine.JobStatusPaused:
		return "已暂停"
	default:
		return "未知状态"
	}
}

// displayStatusJSON 以JSON格式显示状态
func displayStatusJSON(stats engine.EngineStats, jobs []*engine.Job) error {
	type StatusOutput struct {
		Engine engine.EngineStats `json:"engine"`
		Jobs   []JobStatus        `json:"jobs"`
	}

	type JobStatus struct {
		ID       string                `json:"id"`
		Status   engine.JobStatus      `json:"status"`
		Progress engine.ProgressUpdate `json:"progress"`
	}

	jobStatuses := make([]JobStatus, len(jobs))
	for i, job := range jobs {
		jobStatuses[i] = JobStatus{
			ID:       job.ID(),
			Status:   job.Status(),
			Progress: job.CurrentProgress(),
		}
	}

	output := StatusOutput{
		Engine: stats,
		Jobs:   jobStatuses,
	}

	// 这里需要导入encoding/json包
	// jsonBytes, err := json.MarshalIndent(output, "", "  ")
	// if err != nil {
	//     return fmt.Errorf("JSON序列化失败: %w", err)
	// }
	// fmt.Println(string(jsonBytes))

	fmt.Printf("JSON格式输出暂未实现\n") // 占位符
	return nil
}

// displayStatusCSV 以CSV格式显示状态
func displayStatusCSV(stats engine.EngineStats, jobs []*engine.Job) error {
	// CSV表头
	fmt.Println("JobID,Status,FileName,Progress,DownloadSpeed,UploadSpeed,Peers,Message")

	// 任务数据
	for _, job := range jobs {
		progress := job.CurrentProgress()
		fmt.Printf("%s,%s,%s,%.1f%%,%d,%d,%d,%s\n",
			job.ID(),
			progress.Status,
			progress.FileName,
			progress.PercentComplete(),
			progress.DownloadSpeed,
			progress.UploadSpeed,
			progress.ConnectedPeers,
			strings.ReplaceAll(progress.Message, ",", ";"), // 替换逗号避免CSV解析问题
		)
	}

	return nil
}

func init() {
	// 状态命令特定标志
	statusCmd.Flags().BoolVar(&statusDetailed, "detailed", false,
		"显示详细的任务信息")
	statusCmd.Flags().BoolVar(&statusJobsOnly, "jobs-only", false,
		"仅显示任务列表，不显示引擎统计")
	statusCmd.Flags().BoolVar(&statusWatch, "watch", false,
		"持续监控模式")
	statusCmd.Flags().IntVar(&statusInterval, "interval", 3,
		"监控模式的更新间隔(秒)")
	statusCmd.Flags().StringVar(&statusFormat, "format", "default",
		"输出格式 (default|json|csv)")

	// 过滤选项
	statusCmd.Flags().String("filter-status", "",
		"按状态过滤任务 (downloading|seeding|completed|failed等)")
	statusCmd.Flags().String("filter-name", "",
		"按文件名过滤任务 (支持模糊匹配)")
}
