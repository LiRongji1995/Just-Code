package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"goflux/engine"
)

// downloadCmd 处理文件下载
var downloadCmd = &cobra.Command{
	Use:   "download <meta-file-path>",
	Short: "下载指定的.meta文件描述的文件",
	Long: `根据提供的.meta元数据文件下载对应的文件。
.meta文件包含了文件的哈希信息、分片信息和网络节点信息。

下载过程:
  1. 解析.meta元数据文件
  2. 连接到Tracker服务器和DHT网络
  3. 发现并连接到其他Peer节点
  4. 并行下载文件分片
  5. 验证完整性并组装最终文件

示例:
  p2p-downloader download movie.meta
  p2p-downloader download --output ./downloads --peers 100 document.meta
  p2p-downloader download --tracker http://tracker.example.com file.meta
  p2p-downloader download --download-limit 5MB/s large-file.meta`,
	Args: cobra.ExactArgs(1),
	RunE: runDownload,
}

// 下载命令特定的选项
var (
	outputDir      string
	resumeMode     bool
	privateMode    bool
	sequentialMode bool
)

// runDownload 是download命令的核心实现
func runDownload(cmd *cobra.Command, args []string) error {
	metaFilePath := args[0]

	// 1. 验证输入文件
	if err := validateMetaFile(metaFilePath); err != nil {
		return err
	}

	// 2. 构建引擎配置
	config, err := buildEngineConfigForDownload()
	if err != nil {
		return fmt.Errorf("❌ 配置错误: %w", err)
	}

	// 3. 显示配置信息
	printDownloadConfig(config, metaFilePath)

	// 4. 创建引擎实例
	fmt.Printf("🔧 初始化P2P引擎...\n")
	eng, err := engine.NewEngine(config)
	if err != nil {
		return fmt.Errorf("❌ 引擎初始化失败: %w", err)
	}
	defer func() {
		fmt.Printf("\n🔄 正在关闭引擎...\n")
		eng.Shutdown()
	}()

	// 5. 确定输出目录
	finalOutputDir := determineOutputDir(outputDir, config.WorkingDir)

	// 6. 创建下载任务
	fmt.Printf("📦 创建下载任务: %s\n", filepath.Base(metaFilePath))
	job, err := eng.CreateDownloadJob(metaFilePath, finalOutputDir)
	if err != nil {
		return fmt.Errorf("❌ 创建任务失败: %w", err)
	}

	fmt.Printf("🆔 任务ID: %s\n", job.ID())
	fmt.Printf("📁 输出目录: %s\n\n", finalOutputDir)

	// 7. 设置信号处理，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\n🛑 接收到信号 %s，正在取消任务...\n", sig)
		job.Cancel()
		cancel()
	}()

	// 8. 处理任务状态更新 - 这是核心逻辑
	return handleJobEvents(ctx, job)
}

// validateMetaFile 验证元数据文件
func validateMetaFile(metaFilePath string) error {
	// 验证文件扩展名
	if !strings.HasSuffix(metaFilePath, ".meta") {
		return fmt.Errorf("❌ 错误: 请提供有效的.meta文件 (当前: %s)", metaFilePath)
	}

	// 检查文件是否存在
	stat, err := os.Stat(metaFilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("❌ 错误: 找不到文件 %s", metaFilePath)
	}
	if err != nil {
		return fmt.Errorf("❌ 错误: 无法访问文件 %s: %w", metaFilePath, err)
	}

	// 检查是否为目录
	if stat.IsDir() {
		return fmt.Errorf("❌ 错误: %s 是目录，请提供文件路径", metaFilePath)
	}

	// 检查文件大小
	if stat.Size() == 0 {
		return fmt.Errorf("❌ 错误: 元数据文件为空: %s", metaFilePath)
	}

	// 检查文件权限
	file, err := os.Open(metaFilePath)
	if err != nil {
		return fmt.Errorf("❌ 错误: 无法读取文件 %s: %w", metaFilePath, err)
	}
	file.Close()

	return nil
}

// buildEngineConfigForDownload 构建下载任务的引擎配置
func buildEngineConfigForDownload() (engine.Config, error) {
	config := engine.DefaultConfig()

	// 从viper配置更新
	config.ListenPort = viper.GetInt("listen_port")
	config.MaxPeers = viper.GetInt("max_peers")
	config.EnableDHT = viper.GetBool("enable_dht")
	config.WorkingDir = viper.GetString("working_dir")
	config.TrackerURLs = viper.GetStringSlice("tracker_urls")
	config.LogLevel = viper.GetString("log_level")
	config.PieceSize = viper.GetInt("piece_size")
	config.MaxConcurrentDownloads = viper.GetInt("max_concurrent_downloads")

	// 解析超时配置
	if connTimeout := viper.GetString("conn_timeout"); connTimeout != "" {
		if timeout, err := parseDuration(connTimeout); err == nil {
			config.ConnTimeout = timeout
		}
	}
	if reqTimeout := viper.GetString("request_timeout"); reqTimeout != "" {
		if timeout, err := parseDuration(reqTimeout); err == nil {
			config.RequestTimeout = timeout
		}
	}

	// 从命令行标志更新
	if len(trackerURLs) > 0 {
		config.TrackerURLs = append(config.TrackerURLs, trackerURLs...)
	}

	// 解析带宽限制
	if downloadLimit != "" {
		limit, err := parseSpeedLimit(downloadLimit)
		if err != nil {
			return config, fmt.Errorf("无效的下载限制: %w", err)
		}
		config.DownloadLimit = limit
	}

	if uploadLimit != "" {
		limit, err := parseSpeedLimit(uploadLimit)
		if err != nil {
			return config, fmt.Errorf("无效的上传限制: %w", err)
		}
		config.UploadLimit = limit
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return config, err
	}

	return config, nil
}

// printDownloadConfig 显示下载配置信息
func printDownloadConfig(config engine.Config, metaFile string) {
	fmt.Printf("📋 下载配置\n")
	fmt.Printf("============\n")
	fmt.Printf("📄 元数据文件: %s\n", filepath.Base(metaFile))
	fmt.Printf("🌐 监听端口: %d\n", config.ListenPort)
	fmt.Printf("👥 最大连接: %d\n", config.MaxPeers)
	fmt.Printf("🔍 DHT网络: %v\n", config.EnableDHT)

	if len(config.TrackerURLs) > 0 {
		fmt.Printf("📡 Tracker服务器:\n")
		for i, tracker := range config.TrackerURLs {
			fmt.Printf("   %d. %s\n", i+1, tracker)
		}
	}

	if config.DownloadLimit > 0 {
		fmt.Printf("📥 下载限速: %s/s\n", engine.FormatBytes(config.DownloadLimit))
	} else {
		fmt.Printf("📥 下载限速: 无限制\n")
	}

	if config.UploadLimit > 0 {
		fmt.Printf("📤 上传限速: %s/s\n", engine.FormatBytes(config.UploadLimit))
	} else {
		fmt.Printf("📤 上传限速: 无限制\n")
	}

	fmt.Printf("🔧 并发下载: %d\n", config.MaxConcurrentDownloads)
	fmt.Printf("📦 分片大小: %s\n", engine.FormatBytes(int64(config.PieceSize)))
	fmt.Println()
}

// determineOutputDir 确定最终的输出目录
func determineOutputDir(cmdOutputDir, workingDir string) string {
	if cmdOutputDir != "" {
		if filepath.IsAbs(cmdOutputDir) {
			return cmdOutputDir
		}
		return filepath.Join(workingDir, cmdOutputDir)
	}
	return workingDir
}

func init() {
	// 下载命令特定标志
	downloadCmd.Flags().StringVarP(&outputDir, "output", "o", "",
		"输出目录 (默认使用工作目录)")
	downloadCmd.Flags().BoolVarP(&resumeMode, "resume", "r", false,
		"尝试续传下载")
	downloadCmd.Flags().BoolVar(&privateMode, "private", false,
		"私有模式下载 (不连接DHT)")
	downloadCmd.Flags().BoolVar(&sequentialMode, "sequential", false,
		"顺序下载模式 (适合流媒体)")

	// 高级选项
	downloadCmd.Flags().Int("timeout", 0,
		"下载超时时间(秒), 0表示无限制")
	downloadCmd.Flags().Int("retries", 0,
		"最大重试次数 (0使用默认配置)")
	downloadCmd.Flags().Bool("verify", true,
		"下载完成后验证文件完整性")
	downloadCmd.Flags().Bool("seed-after", false,
		"下载完成后自动开始做种")
}
