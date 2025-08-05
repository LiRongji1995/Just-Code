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

// createCmd 处理创建种子和做种
var createCmd = &cobra.Command{
	Use:   "create <file-path>",
	Short: "为指定文件创建.meta文件并开始做种",
	Long: `为本地文件创建.meta元数据文件并开始做种，
使其他用户能够从你这里下载该文件。

创建过程:
  1. 分析文件并计算各分片的哈希值
  2. 生成包含文件信息的.meta元数据文件
  3. 连接到Tracker服务器和DHT网络
  4. 广播文件可用性信息
  5. 等待其他节点连接并提供文件服务

生成的.meta文件将保存在与原文件相同的目录中。

示例:
  p2p-downloader create /path/to/file.txt
  p2p-downloader create --port 6882 large-file.zip
  p2p-downloader create --tracker http://my-tracker.com document.pdf
  p2p-downloader create --private --comment "内部文件" secret.zip`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

// 创建命令特定的选项
var (
	createPrivate bool
	createComment string
	announceURLs  []string
	pieceLength   int
	metaOutputDir string
	seedOnly      bool
	noSeed        bool
)

// runCreate 是create命令的核心实现
func runCreate(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// 1. 验证输入文件
	if err := validateSourceFile(filePath); err != nil {
		return err
	}

	// 2. 构建引擎配置
	config, err := buildEngineConfigForCreate()
	if err != nil {
		return fmt.Errorf("❌ 配置错误: %w", err)
	}

	// 3. 显示创建配置信息
	printCreateConfig(config, filePath)

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

	// 5. 创建做种任务
	fmt.Printf("🌱 创建做种任务: %s\n", filepath.Base(filePath))

	createMeta := !seedOnly // 如果不是仅做种模式，则需要创建meta文件
	job, err := eng.CreateSeedJob(filePath, createMeta)
	if err != nil {
		return fmt.Errorf("❌ 创建做种任务失败: %w", err)
	}

	fmt.Printf("🆔 任务ID: %s\n", job.ID())
	if createMeta {
		metaPath := getMetaFilePath(filePath)
		fmt.Printf("📄 元数据文件: %s\n", metaPath)
	}
	fmt.Println()

	// 6. 如果只是创建meta文件而不做种，等待创建完成后退出
	if noSeed {
		return waitForMetaCreation(job)
	}

	// 7. 设置信号处理，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\n🛑 接收到信号 %s，正在停止做种...\n", sig)
		job.Cancel()
		cancel()
	}()

	// 8. 处理任务状态更新
	return handleJobEvents(ctx, job)
}

// validateSourceFile 验证源文件
func validateSourceFile(filePath string) error {
	// 检查文件是否存在
	stat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("❌ 错误: 找不到文件 %s", filePath)
	}
	if err != nil {
		return fmt.Errorf("❌ 错误: 无法访问文件 %s: %w", filePath, err)
	}

	// 检查是否为目录
	if stat.IsDir() {
		return fmt.Errorf("❌ 错误: %s 是目录，当前版本不支持目录做种", filePath)
	}

	// 检查文件大小
	if stat.Size() == 0 {
		return fmt.Errorf("❌ 错误: 文件为空: %s", filePath)
	}

	// 检查文件权限
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("❌ 错误: 无法读取文件 %s: %w", filePath, err)
	}
	file.Close()

	// 检查meta文件是否已存在
	metaPath := getMetaFilePath(filePath)
	if !seedOnly {
		if _, err := os.Stat(metaPath); err == nil {
			fmt.Printf("⚠️  警告: 元数据文件已存在: %s\n", metaPath)
			fmt.Printf("   将覆盖现有文件。按Ctrl+C取消，或按Enter继续...\n")

			// 等待用户确认
			var input string
			fmt.Scanln(&input)
		}
	}

	return nil
}

// buildEngineConfigForCreate 构建创建任务的引擎配置
func buildEngineConfigForCreate() (engine.Config, error) {
	config := engine.DefaultConfig()

	// 从viper配置更新
	config.ListenPort = viper.GetInt("listen_port")
	config.MaxPeers = viper.GetInt("max_peers")
	config.EnableDHT = viper.GetBool("enable_dht") && !createPrivate // 私有种子不使用DHT
	config.WorkingDir = viper.GetString("working_dir")
	config.TrackerURLs = viper.GetStringSlice("tracker_urls")
	config.LogLevel = viper.GetString("log_level")

	// 分片大小配置
	if pieceLength > 0 {
		config.PieceSize = pieceLength
	} else {
		config.PieceSize = viper.GetInt("piece_size")
	}

	// 从命令行标志更新
	if len(trackerURLs) > 0 {
		config.TrackerURLs = append(config.TrackerURLs, trackerURLs...)
	}
	if len(announceURLs) > 0 {
		config.TrackerURLs = append(config.TrackerURLs, announceURLs...)
	}

	// 解析带宽限制
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

// printCreateConfig 显示创建配置信息
func printCreateConfig(config engine.Config, filePath string) {
	stat, _ := os.Stat(filePath)

	fmt.Printf("📋 创建配置\n")
	fmt.Printf("============\n")
	fmt.Printf("📄 源文件: %s\n", filepath.Base(filePath))
	fmt.Printf("📏 文件大小: %s\n", engine.FormatBytes(stat.Size()))
	fmt.Printf("🌐 监听端口: %d\n", config.ListenPort)
	fmt.Printf("👥 最大连接: %d\n", config.MaxPeers)

	if createPrivate {
		fmt.Printf("🔒 私有种子: 是\n")
		fmt.Printf("🔍 DHT网络: 禁用\n")
	} else {
		fmt.Printf("🔒 私有种子: 否\n")
		fmt.Printf("🔍 DHT网络: %v\n", config.EnableDHT)
	}

	if len(config.TrackerURLs) > 0 {
		fmt.Printf("📡 Tracker服务器:\n")
		for i, tracker := range config.TrackerURLs {
			fmt.Printf("   %d. %s\n", i+1, tracker)
		}
	} else if createPrivate {
		fmt.Printf("⚠️  警告: 私有种子但未配置Tracker服务器\n")
	}

	if config.UploadLimit > 0 {
		fmt.Printf("📤 上传限速: %s/s\n", engine.FormatBytes(config.UploadLimit))
	} else {
		fmt.Printf("📤 上传限速: 无限制\n")
	}

	fmt.Printf("📦 分片大小: %s\n", engine.FormatBytes(int64(config.PieceSize)))

	if createComment != "" {
		fmt.Printf("💬 备注: %s\n", createComment)
	}

	if noSeed {
		fmt.Printf("🔧 模式: 仅创建元数据文件\n")
	} else if seedOnly {
		fmt.Printf("🔧 模式: 仅做种 (使用现有元数据)\n")
	} else {
		fmt.Printf("🔧 模式: 创建元数据并做种\n")
	}

	fmt.Println()
}

// getMetaFilePath 获取meta文件路径
func getMetaFilePath(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// 如果指定了meta输出目录
	if metaOutputDir != "" {
		if filepath.IsAbs(metaOutputDir) {
			dir = metaOutputDir
		} else {
			dir = filepath.Join(dir, metaOutputDir)
		}
	}

	return filepath.Join(dir, name+".meta")
}

// waitForMetaCreation 等待meta文件创建完成
func waitForMetaCreation(job *engine.Job) error {
	fmt.Printf("📝 正在创建元数据文件，请稍候...\n")

	for {
		select {
		case progress := <-job.Progress():
			if progress.Status == engine.JobStatusCompleted {
				fmt.Printf("✅ 元数据文件创建完成！\n")
				return nil
			} else if progress.Status == engine.JobStatusFailed {
				return fmt.Errorf("❌ 元数据文件创建失败: %s", progress.Message)
			}

			// 显示创建进度
			fmt.Printf("📊 %s\n", progress.Message)

		case err := <-job.Errors():
			if err.Fatal {
				return fmt.Errorf("❌ 创建失败: %s", err.Message)
			}
			fmt.Printf("⚠️  警告: %s\n", err.Message)

		case <-job.Done():
			return nil
		}
	}
}

func init() {
	// 创建命令特定标志
	createCmd.Flags().BoolVar(&createPrivate, "private", false,
		"创建私有种子 (不使用DHT网络)")
	createCmd.Flags().StringVar(&createComment, "comment", "",
		"种子备注信息")
	createCmd.Flags().StringSliceVar(&announceURLs, "announce", []string{},
		"额外的Tracker URL列表")
	createCmd.Flags().IntVar(&pieceLength, "piece-size", 0,
		"分片大小(字节), 0使用默认值")
	createCmd.Flags().StringVar(&metaOutputDir, "meta-dir", "",
		"元数据文件输出目录 (默认与源文件同目录)")
	createCmd.Flags().BoolVar(&seedOnly, "seed-only", false,
		"仅做种模式 (使用现有的.meta文件)")
	createCmd.Flags().BoolVar(&noSeed, "no-seed", false,
		"仅创建.meta文件，不开始做种")

	// 高级选项
	createCmd.Flags().Int("announce-interval", 0,
		"Tracker通告间隔(秒), 0使用默认值")
	createCmd.Flags().Bool("optimize-piece-size", false,
		"根据文件大小自动优化分片大小")
}
