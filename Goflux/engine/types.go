// types.go - 引擎相关的类型定义
package engine

import "time"

// EngineStats 包含引擎的全局统计信息
type EngineStats struct {
	ActiveJobs      int           `json:"active_jobs"`      // 活跃任务数
	TotalDownloaded int64         `json:"total_downloaded"` // 总下载量
	TotalUploaded   int64         `json:"total_uploaded"`   // 总上传量
	ConnectedPeers  int           `json:"connected_peers"`  // 总连接数
	Uptime          time.Duration `json:"uptime"`           // 运行时间

	// 新增统计项
	CompletedJobs     int     `json:"completed_jobs"`     // 已完成任务数
	FailedJobs        int     `json:"failed_jobs"`        // 失败任务数
	AverageSpeed      int64   `json:"average_speed"`      // 平均下载速度
	PeakSpeed         int64   `json:"peak_speed"`         // 峰值速度
	TotalConnections  int64   `json:"total_connections"`  // 历史连接总数
	ConnectionSuccess float64 `json:"connection_success"` // 连接成功率

	// 资源使用统计
	MemoryUsage int64   `json:"memory_usage"` // 内存使用量
	DiskUsage   int64   `json:"disk_usage"`   // 磁盘使用量
	CPUUsage    float64 `json:"cpu_usage"`    // CPU使用率
	NetworkIn   int64   `json:"network_in"`   // 网络接收字节数
	NetworkOut  int64   `json:"network_out"`  // 网络发送字节数
}

// JobMetrics 任务级别的统计指标
type JobMetrics struct {
	JobID           string        `json:"job_id"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Duration        time.Duration `json:"duration"`
	TotalSize       int64         `json:"total_size"`
	DownloadedSize  int64         `json:"downloaded_size"`
	UploadedSize    int64         `json:"uploaded_size"`
	AverageSpeed    int64         `json:"average_speed"`
	PeakSpeed       int64         `json:"peak_speed"`
	PeerConnections int           `json:"peer_connections"`
	FailedPieces    int           `json:"failed_pieces"`
	RetryCount      int           `json:"retry_count"`
	Efficiency      float64       `json:"efficiency"` // 实际下载量/传输量
}

// NetworkStats 网络层统计
type NetworkStats struct {
	TotalConnections  int64         `json:"total_connections"`
	ActiveConnections int           `json:"active_connections"`
	FailedConnections int64         `json:"failed_connections"`
	AvgConnectionTime time.Duration `json:"avg_connection_time"`
	BytesReceived     int64         `json:"bytes_received"`
	BytesSent         int64         `json:"bytes_sent"`
	PacketsReceived   int64         `json:"packets_received"`
	PacketsSent       int64         `json:"packets_sent"`
	ErrorCount        int64         `json:"error_count"`
	TimeoutCount      int64         `json:"timeout_count"`
}

// PeerStats Peer级别统计
type PeerStats struct {
	PeerID           string        `json:"peer_id"`
	Address          string        `json:"address"`
	ConnectedAt      time.Time     `json:"connected_at"`
	LastActivity     time.Time     `json:"last_activity"`
	BytesDownloaded  int64         `json:"bytes_downloaded"`
	BytesUploaded    int64         `json:"bytes_uploaded"`
	CurrentDownSpeed int64         `json:"current_down_speed"`
	CurrentUpSpeed   int64         `json:"current_up_speed"`
	PeakDownSpeed    int64         `json:"peak_down_speed"`
	PeakUpSpeed      int64         `json:"peak_up_speed"`
	PiecesReceived   int           `json:"pieces_received"`
	PiecesSent       int           `json:"pieces_sent"`
	FailedRequests   int           `json:"failed_requests"`
	Reliability      float64       `json:"reliability"` // 成功率
	Ping             time.Duration `json:"ping"`        // 延迟
}

// TrackerStats Tracker统计
type TrackerStats struct {
	URL              string        `json:"url"`
	Status           string        `json:"status"` // "active", "error", "timeout"
	LastAnnounce     time.Time     `json:"last_announce"`
	NextAnnounce     time.Time     `json:"next_announce"`
	AnnounceInterval time.Duration `json:"announce_interval"`
	Seeders          int           `json:"seeders"`
	Leechers         int           `json:"leechers"`
	Downloaded       int           `json:"downloaded"` // 完成下载的客户端数
	ResponseTime     time.Duration `json:"response_time"`
	ErrorCount       int           `json:"error_count"`
	SuccessCount     int           `json:"success_count"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Timestamp        time.Time `json:"timestamp"`
	CPUUsagePercent  float64   `json:"cpu_usage_percent"`
	MemoryUsageMB    int64     `json:"memory_usage_mb"`
	DiskIOReadMBps   float64   `json:"disk_io_read_mbps"`
	DiskIOWriteMBps  float64   `json:"disk_io_write_mbps"`
	NetworkInMBps    float64   `json:"network_in_mbps"`
	NetworkOutMBps   float64   `json:"network_out_mbps"`
	ActiveGoroutines int       `json:"active_goroutines"`
	HeapSizeMB       int64     `json:"heap_size_mb"`
	GCPauseMicros    int64     `json:"gc_pause_micros"`
}

// QueueStats 队列统计
type QueueStats struct {
	PendingTasks     int           `json:"pending_tasks"`
	ProcessingTasks  int           `json:"processing_tasks"`
	CompletedTasks   int           `json:"completed_tasks"`
	FailedTasks      int           `json:"failed_tasks"`
	AverageWaitTime  time.Duration `json:"average_wait_time"`
	AverageTaskTime  time.Duration `json:"average_task_time"`
	QueueCapacity    int           `json:"queue_capacity"`
	QueueUtilization float64       `json:"queue_utilization"`
}

// ErrorStats 错误统计
type ErrorStats struct {
	NetworkErrors    int64   `json:"network_errors"`
	IOErrors         int64   `json:"io_errors"`
	ProtocolErrors   int64   `json:"protocol_errors"`
	TimeoutErrors    int64   `json:"timeout_errors"`
	ValidationErrors int64   `json:"validation_errors"`
	PermissionErrors int64   `json:"permission_errors"`
	ResourceErrors   int64   `json:"resource_errors"`
	UserErrors       int64   `json:"user_errors"`
	TotalErrors      int64   `json:"total_errors"`
	ErrorRate        float64 `json:"error_rate"` // 错误率
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status          string    `json:"status"` // "healthy", "warning", "critical"
	LastCheck       time.Time `json:"last_check"`
	Issues          []string  `json:"issues"`     // 当前问题列表
	CPUHealth       string    `json:"cpu_health"` // "good", "warning", "critical"
	MemoryHealth    string    `json:"memory_health"`
	DiskHealth      string    `json:"disk_health"`
	NetworkHealth   string    `json:"network_health"`
	OverallScore    int       `json:"overall_score"`   // 0-100分
	Recommendations []string  `json:"recommendations"` // 优化建议
}

// ConfigurationInfo 配置信息
type ConfigurationInfo struct {
	Version        string        `json:"version"`
	BuildTime      time.Time     `json:"build_time"`
	MaxPeers       int           `json:"max_peers"`
	MaxConnections int           `json:"max_connections"`
	PieceSize      int           `json:"piece_size"`
	DownloadLimit  int64         `json:"download_limit"`
	UploadLimit    int64         `json:"upload_limit"`
	ConnTimeout    time.Duration `json:"conn_timeout"`
	RequestTimeout time.Duration `json:"request_timeout"`
	MaxRetries     int           `json:"max_retries"`
	WorkingDir     string        `json:"working_dir"`
	TempDir        string        `json:"temp_dir"`
	EnableDHT      bool          `json:"enable_dht"`
	EnableLogging  bool          `json:"enable_logging"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	OS              string    `json:"os"`
	Architecture    string    `json:"architecture"`
	CPUs            int       `json:"cpus"`
	TotalMemoryMB   int64     `json:"total_memory_mb"`
	AvailMemoryMB   int64     `json:"avail_memory_mb"`
	DiskSpaceGB     int64     `json:"disk_space_gb"`
	FreeDiskSpaceGB int64     `json:"free_disk_space_gb"`
	Hostname        string    `json:"hostname"`
	StartTime       time.Time `json:"start_time"`
	GoVersion       string    `json:"go_version"`
	RuntimeVersion  string    `json:"runtime_version"`
}
