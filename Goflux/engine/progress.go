package engine

import (
	"fmt"
	"time"
)

// ProgressUpdate 包含任务进度的详细信息
// 设计思想：提供足够详细的信息，让UI可以自由选择展示方式
type ProgressUpdate struct {
	JobID  string    `json:"job_id"` // 任务唯一标识
	Status JobStatus `json:"status"` // 当前状态

	// 文件信息
	FileName       string `json:"file_name"`       // 文件名
	TotalSize      int64  `json:"total_size"`      // 文件总大小
	DownloadedSize int64  `json:"downloaded_size"` // 已下载大小
	UploadedSize   int64  `json:"uploaded_size"`   // 已上传大小

	// 分片信息
	TotalPieces     int `json:"total_pieces"`     // 总分片数
	CompletedPieces int `json:"completed_pieces"` // 已完成分片数
	ActivePieces    int `json:"active_pieces"`    // 正在下载的分片数
	PendingPieces   int `json:"pending_pieces"`   // 等待下载的分片数

	// 网络状态
	ConnectedPeers int `json:"connected_peers"` // 已连接Peer数
	ActivePeers    int `json:"active_peers"`    // 活跃Peer数（正在传输）
	SeedPeers      int `json:"seed_peers"`      // 种子节点数
	LeechPeers     int `json:"leech_peers"`     // 下载节点数

	// 速度统计
	DownloadSpeed int64 `json:"download_speed"` // 下载速度 (bytes/s)
	UploadSpeed   int64 `json:"upload_speed"`   // 上传速度 (bytes/s)
	AverageSpeed  int64 `json:"average_speed"`  // 平均下载速度 (bytes/s)

	// 时间信息
	ElapsedTime   time.Duration `json:"elapsed_time"`   // 已用时间
	EstimatedTime time.Duration `json:"estimated_time"` // 预计剩余时间
	StartTime     time.Time     `json:"start_time"`     // 开始时间

	// 额外信息
	Message   string    `json:"message"`   // 状态描述消息
	Timestamp time.Time `json:"timestamp"` // 更新时间戳

	// 高级统计
	Ratio        float64 `json:"ratio"`        // 分享率 (上传/下载)
	Availability float64 `json:"availability"` // 可用性 (0-1)
	WastedBytes  int64   `json:"wasted_bytes"` // 浪费的字节数
}

// PercentComplete 计算进度百分比
func (p *ProgressUpdate) PercentComplete() float64 {
	if p.TotalSize == 0 {
		return 0
	}
	return float64(p.DownloadedSize) / float64(p.TotalSize) * 100
}

// PieceProgress 计算分片进度百分比
func (p *ProgressUpdate) PieceProgress() float64 {
	if p.TotalPieces == 0 {
		return 0
	}
	return float64(p.CompletedPieces) / float64(p.TotalPieces) * 100
}

// IsComplete 检查任务是否完成
func (p *ProgressUpdate) IsComplete() bool {
	return p.Status == JobStatusCompleted
}

// IsActive 检查任务是否活跃（正在进行）
func (p *ProgressUpdate) IsActive() bool {
	return p.Status == JobStatusDownloading || p.Status == JobStatusSeeding
}

// RemainingSize 计算剩余下载大小
func (p *ProgressUpdate) RemainingSize() int64 {
	return p.TotalSize - p.DownloadedSize
}

// FormatSpeed 格式化速度为人类可读格式
func (p *ProgressUpdate) FormatSpeed(speed int64) string {
	return FormatBytesPerSecond(speed)
}

// FormatSize 格式化大小为人类可读格式
func (p *ProgressUpdate) FormatSize(size int64) string {
	return FormatBytes(size)
}

// JobError 表示任务执行过程中的错误
type JobError struct {
	JobID     string    `json:"job_id"`    // 任务ID
	Type      string    `json:"type"`      // 错误类型
	Message   string    `json:"message"`   // 错误消息
	Fatal     bool      `json:"fatal"`     // 是否为致命错误
	Timestamp time.Time `json:"timestamp"` // 错误时间
	Context   string    `json:"context"`   // 错误上下文
	Code      int       `json:"code"`      // 错误代码
}

// Error 实现error接口
func (e *JobError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("[%s] %s (%s): %s", e.Type, e.JobID, e.Context, e.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.JobID, e.Message)
}

// String 返回错误的详细字符串表示
func (e *JobError) String() string {
	fatalStr := "non-fatal"
	if e.Fatal {
		fatalStr = "FATAL"
	}

	return fmt.Sprintf("JobError{ID:%s, Type:%s, Fatal:%s, Code:%d, Message:%s, Time:%s}",
		e.JobID, e.Type, fatalStr, e.Code, e.Message, e.Timestamp.Format("15:04:05"))
}

// 错误类型常量
const (
	ErrorTypeNetwork    = "network"    // 网络错误
	ErrorTypeIO         = "io"         // I/O错误
	ErrorTypeProtocol   = "protocol"   // 协议错误
	ErrorTypeValidation = "validation" // 验证错误
	ErrorTypeTimeout    = "timeout"    // 超时错误
	ErrorTypePermission = "permission" // 权限错误
	ErrorTypeResource   = "resource"   // 资源错误
	ErrorTypeUser       = "user"       // 用户操作错误
)

// NewJobError 创建新的任务错误
func NewJobError(jobID, errorType, message string, fatal bool) *JobError {
	return &JobError{
		JobID:     jobID,
		Type:      errorType,
		Message:   message,
		Fatal:     fatal,
		Timestamp: time.Now(),
		Code:      0,
	}
}

// NewJobErrorWithCode 创建带错误代码的任务错误
func NewJobErrorWithCode(jobID, errorType, message string, fatal bool, code int) *JobError {
	return &JobError{
		JobID:     jobID,
		Type:      errorType,
		Message:   message,
		Fatal:     fatal,
		Timestamp: time.Now(),
		Code:      code,
	}
}

// NewJobErrorWithContext 创建带上下文的任务错误
func NewJobErrorWithContext(jobID, errorType, message, context string, fatal bool) *JobError {
	return &JobError{
		JobID:     jobID,
		Type:      errorType,
		Message:   message,
		Fatal:     fatal,
		Timestamp: time.Now(),
		Context:   context,
		Code:      0,
	}
}

// 辅助函数：格式化字节数
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 辅助函数：格式化速度
func FormatBytesPerSecond(bytesPerSec int64) string {
	if bytesPerSec == 0 {
		return "0 B/s"
	}
	return FormatBytes(bytesPerSec) + "/s"
}

// PeerInfo 表示Peer节点信息
type PeerInfo struct {
	ID        string    `json:"id"`         // Peer ID
	Address   string    `json:"address"`    // IP地址和端口
	Client    string    `json:"client"`     // 客户端类型
	Progress  float64   `json:"progress"`   // 该Peer的完成度
	DownSpeed int64     `json:"down_speed"` // 从该Peer的下载速度
	UpSpeed   int64     `json:"up_speed"`   // 向该Peer的上传速度
	Connected time.Time `json:"connected"`  // 连接时间
}

// TrackerInfo 表示Tracker信息
type TrackerInfo struct {
	URL          string        `json:"url"`           // Tracker URL
	Status       string        `json:"status"`        // 状态 (working/error/timeout)
	LastAnnounce time.Time     `json:"last_announce"` // 最后通告时间
	NextAnnounce time.Time     `json:"next_announce"` // 下次通告时间
	Interval     time.Duration `json:"interval"`      // 通告间隔
	Seeders      int           `json:"seeders"`       // 种子数
	Leechers     int           `json:"leechers"`      // 下载者数
	ErrorMsg     string        `json:"error_msg"`     // 错误消息
}

// DetailedProgress 包含更详细的进度信息
type DetailedProgress struct {
	ProgressUpdate

	// 详细信息
	Peers    []PeerInfo    `json:"peers"`    // Peer列表
	Trackers []TrackerInfo `json:"trackers"` // Tracker列表

	// 分片详情
	PieceBitfield []bool `json:"piece_bitfield"` // 分片位图

	// 统计信息
	SessionDownloaded int64 `json:"session_downloaded"` // 本次会话下载量
	SessionUploaded   int64 `json:"session_uploaded"`   // 本次会话上传量
	AllTimeDownloaded int64 `json:"alltime_downloaded"` // 总下载量
	AllTimeUploaded   int64 `json:"alltime_uploaded"`   // 总上传量
}
