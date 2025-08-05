package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	// 全局配置选项
	trackerURLs   []string
	listenPort    int
	maxPeers      int
	uploadLimit   string
	downloadLimit string
	workingDir    string
	enableDHT     bool
	logLevel      string
	configDir     string
)

// rootCmd 是应用的根命令
var rootCmd = &cobra.Command{
	Use:   "p2p-downloader",
	Short: "一个高效的P2P文件下载器",
	Long: `p2p-downloader 是一个基于Go语言开发的P2P文件下载工具。
支持BitTorrent协议，提供快速可靠的分布式文件传输服务。

主要特性:
  • 支持多Tracker和DHT网络
  • 实时进度显示和速度监控  
  • 可配置的并发连接和带宽限制
  • 支持断点续传和任务暂停/恢复
  • 完全模块化的架构设计

架构说明:
  本工具采用引擎-界面分离的设计，核心P2P引擎完全独立于用户界面。
  这使得相同的引擎可以支持CLI、GUI或Web界面等多种前端。`,
	Version: "1.0.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 在每个命令执行前进行全局初始化检查
		return validateGlobalConfig()
	},
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
}

// validateGlobalConfig 验证全局配置
func validateGlobalConfig() error {
	// 验证端口范围
	if listenPort < 1024 || listenPort > 65535 {
		return fmt.Errorf("监听端口必须在1024-65535范围内，当前值: %d", listenPort)
	}

	// 验证最大连接数
	if maxPeers < 1 || maxPeers > 1000 {
		return fmt.Errorf("最大连接数必须在1-1000范围内，当前值: %d", maxPeers)
	}

	// 验证工作目录
	if workingDir != "" {
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			return fmt.Errorf("无法创建工作目录 %s: %w", workingDir, err)
		}
	}

	return nil
}

// initConfig 初始化配置系统
func initConfig() {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 查找默认配置文件位置
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("⚠️  获取用户目录失败: %v\n", err)
			return
		}

		// 设置配置文件搜索路径
		configDir = filepath.Join(home, ".config", "p2p-downloader")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// 确保配置目录存在
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("⚠️  创建配置目录失败: %v\n", err)
		}
	}

	// 设置默认值
	setDefaultValues()

	// 设置环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvPrefix("P2P")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，创建默认配置文件
			if err := createDefaultConfig(); err != nil {
				fmt.Printf("⚠️  创建默认配置文件失败: %v\n", err)
			}
		} else {
			fmt.Printf("⚠️  读取配置文件失败: %v\n", err)
		}
	} else {
		fmt.Printf("📝 使用配置文件: %s\n", viper.ConfigFileUsed())
	}
}

// setDefaultValues 设置配置默认值
func setDefaultValues() {
	home, _ := os.UserHomeDir()
	defaultWorkingDir := filepath.Join(home, "Downloads", "p2p-downloader")

	viper.SetDefault("listen_port", 6881)
	viper.SetDefault("max_peers", 50)
	viper.SetDefault("working_dir", defaultWorkingDir)
	viper.SetDefault("enable_dht", true)
	viper.SetDefault("tracker_urls", []string{})
	viper.SetDefault("upload_limit", "")
	viper.SetDefault("download_limit", "")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("piece_size", 256*1024)
	viper.SetDefault("max_concurrent_downloads", 4)
	viper.SetDefault("conn_timeout", "30s")
	viper.SetDefault("request_timeout", "10s")
	viper.SetDefault("max_retries", 3)
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig() error {
	if configDir == "" {
		return fmt.Errorf("配置目录未设置")
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// 检查文件是否已存在
	if _, err := os.Stat(configFile); err == nil {
		return nil // 文件已存在
	}

	// 创建默认配置内容
	defaultConfig := `# P2P下载器配置文件
# 网络配置
listen_port: 6881
max_peers: 50
enable_dht: true
tracker_urls: []

# 传输配置
upload_limit: ""      # 示例: "1MB/s", 空字符串表示无限制
download_limit: ""    # 示例: "5MB/s", 空字符串表示无限制
piece_size: 262144    # 256KB

# 存储配置
working_dir: "~/Downloads/p2p-downloader"
temp_dir: ""          # 空字符串使用working_dir/.tmp

# 性能配置
max_concurrent_downloads: 4
conn_timeout: "30s"
request_timeout: "10s"
max_retries: 3

# 日志配置
log_level: "info"     # debug, info, warn, error, fatal
enable_log: true
`

	// 写入文件
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	fmt.Printf("✅ 已创建默认配置文件: %s\n", configFile)
	return nil
}

// GetConfigValue 获取配置值的辅助函数
func GetConfigValue(key string) interface{} {
	return viper.Get(key)
}

// SetConfigValue 设置配置值的辅助函数
func SetConfigValue(key string, value interface{}) {
	viper.Set(key, value)
}

// SaveConfig 保存当前配置到文件
func SaveConfig() error {
	return viper.WriteConfig()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局持久化标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"配置文件路径 (默认: $HOME/.config/p2p-downloader/config.yaml)")
	rootCmd.PersistentFlags().StringSliceVar(&trackerURLs, "tracker", []string{},
		"Tracker服务器URL列表")
	rootCmd.PersistentFlags().IntVar(&listenPort, "port", 6881,
		"监听端口")
	rootCmd.PersistentFlags().IntVar(&maxPeers, "peers", 50,
		"最大连接节点数")
	rootCmd.PersistentFlags().StringVar(&workingDir, "working-dir", "",
		"工作目录 (默认: ~/Downloads/p2p-downloader)")
	rootCmd.PersistentFlags().BoolVar(&enableDHT, "dht", true,
		"启用DHT网络")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"日志级别 (debug|info|warn|error|fatal)")

	// 传输相关标志
	rootCmd.PersistentFlags().StringVar(&uploadLimit, "upload-limit", "",
		"上传速度限制 (如: 1MB/s, 500KB/s)")
	rootCmd.PersistentFlags().StringVar(&downloadLimit, "download-limit", "",
		"下载速度限制 (如: 1MB/s, 500KB/s)")

	// 绑定配置
	viper.BindPFlag("listen_port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("max_peers", rootCmd.PersistentFlags().Lookup("peers"))
	viper.BindPFlag("working_dir", rootCmd.PersistentFlags().Lookup("working-dir"))
	viper.BindPFlag("enable_dht", rootCmd.PersistentFlags().Lookup("dht"))
	viper.BindPFlag("tracker_urls", rootCmd.PersistentFlags().Lookup("tracker"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("upload_limit", rootCmd.PersistentFlags().Lookup("upload-limit"))
	viper.BindPFlag("download_limit", rootCmd.PersistentFlags().Lookup("download-limit"))

	// 添加子命令
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(statusCmd)
}
