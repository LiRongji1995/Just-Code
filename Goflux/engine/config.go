package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config 包含引擎的所有配置参数
type Config struct {
	// 网络配置
	ListenPort  int      `json:"listen_port" yaml:"listen_port"`   // 监听端口
	TrackerURLs []string `json:"tracker_urls" yaml:"tracker_urls"` // Tracker服务器列表
	EnableDHT   bool     `json:"enable_dht" yaml:"enable_dht"`     // 是否启用DHT网络
	MaxPeers    int      `json:"max_peers" yaml:"max_peers"`       // 最大连接Peer数量

	// 传输配置
	UploadLimit   int64 `json:"upload_limit" yaml:"upload_limit"`     // 上传速度限制 (bytes/s, 0为无限制)
	DownloadLimit int64 `json:"download_limit" yaml:"download_limit"` // 下载速度限制 (bytes/s, 0为无限制)
	PieceSize     int   `json:"piece_size" yaml:"piece_size"`         // 分片大小 (bytes)

	// 存储配置
	WorkingDir string `json:"working_dir" yaml:"working_dir"` // 工作目录
	TempDir    string `json:"temp_dir" yaml:"temp_dir"`       // 临时文件目录

	// 高级配置
	ConnTimeout      time.Duration `json:"conn_timeout" yaml:"conn_timeout"`           // 连接超时
	RequestTimeout   time.Duration `json:"request_timeout" yaml:"request_timeout"`     // 请求超时
	MaxRetries       int           `json:"max_retries" yaml:"max_retries"`             // 最大重试次数
	HandshakeTimeout time.Duration `json:"handshake_timeout" yaml:"handshake_timeout"` // 握手超时

	// 性能调优
	MaxConcurrentDownloads int `json:"max_concurrent_downloads" yaml:"max_concurrent_downloads"` // 最大并发下载数
	BufferSize             int `json:"buffer_size" yaml:"buffer_size"`                           // I/O缓冲区大小

	// 日志和调试
	LogLevel  string `json:"log_level" yaml:"log_level"`   // 日志级别
	EnableLog bool   `json:"enable_log" yaml:"enable_log"` // 是否启用日志
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	defaultWorkingDir := filepath.Join(homeDir, "Downloads", "p2p-downloader")
	defaultTempDir := filepath.Join(defaultWorkingDir, ".tmp")

	return Config{
		// 网络配置
		ListenPort:  6881,
		TrackerURLs: []string{},
		EnableDHT:   true,
		MaxPeers:    50,

		// 传输配置
		UploadLimit:   0,          // 无限制
		DownloadLimit: 0,          // 无限制
		PieceSize:     256 * 1024, // 256KB

		// 存储配置
		WorkingDir: defaultWorkingDir,
		TempDir:    defaultTempDir,

		// 高级配置
		ConnTimeout:      30 * time.Second,
		RequestTimeout:   10 * time.Second,
		MaxRetries:       3,
		HandshakeTimeout: 5 * time.Second,

		// 性能调优
		MaxConcurrentDownloads: 4,
		BufferSize:             64 * 1024, // 64KB

		// 日志和调试
		LogLevel:  "info",
		EnableLog: true,
	}
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证端口范围
	if c.ListenPort < 1024 || c.ListenPort > 65535 {
		return fmt.Errorf("监听端口必须在1024-65535范围内，当前值: %d", c.ListenPort)
	}

	// 验证最大连接数
	if c.MaxPeers < 1 || c.MaxPeers > 1000 {
		return fmt.Errorf("最大连接数必须在1-1000范围内，当前值: %d", c.MaxPeers)
	}

	// 验证分片大小
	if c.PieceSize < 16*1024 || c.PieceSize > 16*1024*1024 {
		return fmt.Errorf("分片大小必须在16KB-16MB范围内，当前值: %d", c.PieceSize)
	}

	// 验证速度限制
	if c.UploadLimit < 0 || c.DownloadLimit < 0 {
		return fmt.Errorf("速度限制不能为负数")
	}

	// 验证并创建目录
	if err := c.ensureDirectories(); err != nil {
		return fmt.Errorf("目录创建失败: %w", err)
	}

	// 验证超时配置
	if c.ConnTimeout <= 0 || c.RequestTimeout <= 0 || c.HandshakeTimeout <= 0 {
		return fmt.Errorf("超时配置必须大于0")
	}

	// 验证并发下载数
	if c.MaxConcurrentDownloads < 1 || c.MaxConcurrentDownloads > 20 {
		return fmt.Errorf("最大并发下载数必须在1-20范围内，当前值: %d", c.MaxConcurrentDownloads)
	}

	// 验证缓冲区大小
	if c.BufferSize < 4*1024 || c.BufferSize > 1024*1024 {
		return fmt.Errorf("缓冲区大小必须在4KB-1MB范围内，当前值: %d", c.BufferSize)
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("无效的日志级别: %s", c.LogLevel)
	}

	return nil
}

// ensureDirectories 确保必要的目录存在
func (c *Config) ensureDirectories() error {
	directories := []string{c.WorkingDir, c.TempDir}

	for _, dir := range directories {
		if dir == "" {
			continue
		}

		// 检查目录是否存在
		if stat, err := os.Stat(dir); err == nil {
			if !stat.IsDir() {
				return fmt.Errorf("路径 %s 存在但不是目录", dir)
			}
			continue
		}

		// 创建目录
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("无法创建目录 %s: %w", dir, err)
		}
	}

	return nil
}

// Clone 创建配置的深拷贝
func (c *Config) Clone() Config {
	clone := *c

	// 深拷贝切片
	if len(c.TrackerURLs) > 0 {
		clone.TrackerURLs = make([]string, len(c.TrackerURLs))
		copy(clone.TrackerURLs, c.TrackerURLs)
	}

	return clone
}

// AddTracker 添加Tracker URL
func (c *Config) AddTracker(url string) {
	// 检查是否已存在
	for _, existing := range c.TrackerURLs {
		if existing == url {
			return
		}
	}

	c.TrackerURLs = append(c.TrackerURLs, url)
}

// RemoveTracker 移除Tracker URL
func (c *Config) RemoveTracker(url string) {
	for i, existing := range c.TrackerURLs {
		if existing == url {
			c.TrackerURLs = append(c.TrackerURLs[:i], c.TrackerURLs[i+1:]...)
			return
		}
	}
}

// SetSpeedLimits 设置速度限制（以MB/s为单位）
func (c *Config) SetSpeedLimits(downloadMBps, uploadMBps float64) {
	const bytesPerMB = 1024 * 1024

	if downloadMBps <= 0 {
		c.DownloadLimit = 0 // 无限制
	} else {
		c.DownloadLimit = int64(downloadMBps * bytesPerMB)
	}

	if uploadMBps <= 0 {
		c.UploadLimit = 0 // 无限制
	} else {
		c.UploadLimit = int64(uploadMBps * bytesPerMB)
	}
}

// GetDownloadLimitMBps 获取下载速度限制（MB/s）
func (c *Config) GetDownloadLimitMBps() float64 {
	if c.DownloadLimit == 0 {
		return 0 // 无限制
	}
	return float64(c.DownloadLimit) / (1024 * 1024)
}

// GetUploadLimitMBps 获取上传速度限制（MB/s）
func (c *Config) GetUploadLimitMBps() float64 {
	if c.UploadLimit == 0 {
		return 0 // 无限制
	}
	return float64(c.UploadLimit) / (1024 * 1024)
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf(`P2P Engine Configuration:
  Network:
    Listen Port: %d
    Max Peers: %d
    Enable DHT: %v
    Trackers: %d configured
  
  Transfer:
    Download Limit: %.1f MB/s
    Upload Limit: %.1f MB/s
    Piece Size: %d KB
  
  Storage:
    Working Dir: %s
    Temp Dir: %s
  
  Performance:
    Max Concurrent Downloads: %d
    Buffer Size: %d KB
    Connection Timeout: %s`,
		c.ListenPort,
		c.MaxPeers,
		c.EnableDHT,
		len(c.TrackerURLs),
		c.GetDownloadLimitMBps(),
		c.GetUploadLimitMBps(),
		c.PieceSize/1024,
		c.WorkingDir,
		c.TempDir,
		c.MaxConcurrentDownloads,
		c.BufferSize/1024,
		c.ConnTimeout,
	)
}
