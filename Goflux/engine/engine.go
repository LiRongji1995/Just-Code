// Package engine 提供了一个与UI完全解耦的P2P下载引擎
// 设计思想：
// 1. 通过channel实现异步事件驱动的状态更新机制
// 2. 所有状态变化通过结构化数据传递，而非直接输出
// 3. 引擎职责单一：专注于P2P网络协议与文件管理
// 4. 外部UI可以自由选择如何展示状态信息
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Engine 是P2P下载引擎的主要接口
// 设计思想：
// 1. 单例模式管理全局网络资源（端口、连接池等）
// 2. 支持多任务并发执行
// 3. 提供统一的任务生命周期管理
type Engine struct {
	config Config
	ctx    context.Context
	cancel context.CancelFunc

	// 任务管理
	jobs   map[string]*Job
	jobsMu sync.RWMutex

	// 网络资源
	listener interface{} // 实际实现时会是具体的网络监听器
	peerPool interface{} // Peer连接池
	tracker  interface{} // Tracker客户端

	// 状态统计
	stats     EngineStats
	statsMu   sync.RWMutex
	startTime time.Time
}

// EngineStats 包含引擎的全局统计信息
type EngineStats struct {
	ActiveJobs      int           `json:"active_jobs"`      // 活跃任务数
	TotalDownloaded int64         `json:"total_downloaded"` // 总下载量
	TotalUploaded   int64         `json:"total_uploaded"`   // 总上传量
	ConnectedPeers  int           `json:"connected_peers"`  // 总连接数
	Uptime          time.Duration `json:"uptime"`           // 运行时间
}

// NewEngine 创建新的P2P引擎实例
// config: 引擎配置参数
// 返回引擎实例和可能的初始化错误
func NewEngine(config Config) (*Engine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		jobs:      make(map[string]*Job),
		stats:     EngineStats{},
		startTime: time.Now(),
	}

	// 初始化网络组件
	if err := engine.initNetwork(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	// 启动后台任务
	go engine.backgroundTasks()

	return engine, nil
}

// CreateDownloadJob 创建新的下载任务
// metaFilePath: .meta文件路径
// outputDir: 输出目录
// 返回任务实例和可能的错误
func (e *Engine) CreateDownloadJob(metaFilePath, outputDir string) (*Job, error) {
	// 生成唯一任务ID
	jobID := fmt.Sprintf("download_%d", time.Now().UnixNano())

	// 创建任务实例
	job := e.createJobInstance(jobID, metaFilePath, outputDir)

	// 注册任务
	e.jobsMu.Lock()
	e.jobs[jobID] = job
	e.jobsMu.Unlock()

	// 启动任务处理协程
	go e.runDownloadJob(job)

	return job, nil
}

// CreateSeedJob 创建新的做种任务
func (e *Engine) CreateSeedJob(filePath string, createMeta bool) (*Job, error) {
	jobID := fmt.Sprintf("seed_%d", time.Now().UnixNano())

	job := e.createJobInstance(jobID, filePath, "")

	e.jobsMu.Lock()
	e.jobs[jobID] = job
	e.jobsMu.Unlock()

	go e.runSeedJob(job, createMeta)

	return job, nil
}

// GetJob 根据ID获取任务
func (e *Engine) GetJob(jobID string) (*Job, bool) {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()
	job, exists := e.jobs[jobID]
	return job, exists
}

// ListJobs 返回所有任务列表
func (e *Engine) ListJobs() []*Job {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()

	jobs := make([]*Job, 0, len(e.jobs))
	for _, job := range e.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// Stats 返回引擎统计信息
func (e *Engine) Stats() EngineStats {
	e.statsMu.RLock()
	defer e.statsMu.RUnlock()

	stats := e.stats
	stats.Uptime = time.Since(e.startTime)
	return stats
}

// Shutdown 优雅关闭引擎
func (e *Engine) Shutdown() error {
	e.cancel()

	// 等待所有任务完成
	e.jobsMu.RLock()
	jobs := make([]*Job, 0, len(e.jobs))
	for _, job := range e.jobs {
		jobs = append(jobs, job)
	}
	e.jobsMu.RUnlock()

	// 取消所有任务
	for _, job := range jobs {
		job.Cancel()
	}

	return nil
}

// createJobInstance 创建任务实例的内部方法
func (e *Engine) createJobInstance(jobID, filePath, outputDir string) *Job {
	jobCtx, jobCancel := context.WithCancel(e.ctx)

	return &Job{
		id:         jobID,
		ctx:        jobCtx,
		cancel:     jobCancel,
		progressCh: make(chan ProgressUpdate, 10), // 带缓冲以避免阻塞
		errorCh:    make(chan *JobError, 10),
		doneCh:     make(chan struct{}),
		status:     JobStatusPending,
		metaFile:   filePath,
		outputDir:  outputDir,
		engine:     e,
		progress: ProgressUpdate{
			JobID:     jobID,
			Status:    JobStatusPending,
			Timestamp: time.Now(),
		},
	}
}

// 私有方法：初始化网络组件
func (e *Engine) initNetwork() error {
	// 实际实现会在这里初始化网络监听器、连接池等
	// 这里只是占位实现
	return nil
}

// 私有方法：后台任务处理
func (e *Engine) backgroundTasks() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateStats()
		case <-e.ctx.Done():
			return
		}
	}
}

// 私有方法：更新统计信息
func (e *Engine) updateStats() {
	e.jobsMu.RLock()
	activeJobs := len(e.jobs)
	e.jobsMu.RUnlock()

	e.statsMu.Lock()
	e.stats.ActiveJobs = activeJobs
	e.statsMu.Unlock()
}

// 私有方法：移除已完成的任务
func (e *Engine) removeJob(jobID string) {
	e.jobsMu.Lock()
	delete(e.jobs, jobID)
	e.jobsMu.Unlock()
}
