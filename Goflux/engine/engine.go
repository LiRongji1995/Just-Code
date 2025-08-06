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
	jobID := generateJobID("download")

	// 创建任务实例
	job := e.createJobInstance(jobID, metaFilePath, outputDir)

	// 注册任务
	e.registerJob(job)

	// 启动任务处理协程
	go e.runDownloadJob(job)

	return job, nil
}

// CreateSeedJob 创建新的做种任务
func (e *Engine) CreateSeedJob(filePath string, createMeta bool) (*Job, error) {
	jobID := generateJobID("seed")

	job := e.createJobInstance(jobID, filePath, "")

	e.registerJob(job)

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

	// 获取所有任务的快照
	jobs := e.getAllJobs()

	// 并发取消所有任务
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(j *Job) {
			defer wg.Done()
			j.Cancel()
		}(job)
	}

	// 等待所有任务完成取消，最多等待30秒
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout: some jobs did not complete within 30 seconds")
	}
}

// runDownloadJob 执行下载任务的主要逻辑
func (e *Engine) runDownloadJob(job *Job) {
	defer e.cleanupJob(job)

	// 下载流程的各个阶段
	stages := []struct {
		name   string
		status JobStatus
		fn     func(*Job) error
	}{
		{"解析元数据文件", JobStatusMetadata, e.parseMetaFile},
		{"连接网络", JobStatusConnecting, e.connectToNetwork},
		{"执行下载", JobStatusDownloading, e.downloadLoop},
		{"验证文件", JobStatusDownloading, e.verifyFile},
	}

	for _, stage := range stages {
		select {
		case <-job.ctx.Done():
			return // 任务已取消
		default:
		}

		// 更新到当前阶段状态
		job.updateStatus(stage.status, fmt.Sprintf("正在%s...", stage.name))

		if err := stage.fn(job); err != nil {
			job.sendError(ErrorTypeNetwork, fmt.Sprintf("%s失败: %v", stage.name, err), true)
			job.updateStatus(JobStatusFailed, fmt.Sprintf("%s失败", stage.name))
			return
		}
	}

	job.updateStatus(JobStatusCompleted, "下载完成")
}

// runSeedJob 执行做种任务的主要逻辑
func (e *Engine) runSeedJob(job *Job, createMeta bool) {
	defer e.cleanupJob(job)

	// 做种准备阶段
	if createMeta {
		job.updateStatus(JobStatusMetadata, "正在创建元数据文件...")
		if err := e.createMetaFile(job); err != nil {
			job.sendError(ErrorTypeIO, fmt.Sprintf("创建元数据文件失败: %v", err), true)
			job.updateStatus(JobStatusFailed, "创建元数据文件失败")
			return
		}
	}

	job.updateStatus(JobStatusMetadata, "正在解析元数据文件...")
	if err := e.parseMetaFile(job); err != nil {
		job.sendError(ErrorTypeValidation, fmt.Sprintf("解析元数据文件失败: %v", err), true)
		job.updateStatus(JobStatusFailed, "解析元数据文件失败")
		return
	}

	if err := e.verifyLocalFile(job); err != nil {
		job.sendError(ErrorTypeValidation, fmt.Sprintf("本地文件验证失败: %v", err), true)
		job.updateStatus(JobStatusFailed, "本地文件验证失败")
		return
	}

	job.updateStatus(JobStatusConnecting, "正在连接网络...")
	if err := e.announceToTracker(job, true); err != nil {
		job.sendError(ErrorTypeNetwork, fmt.Sprintf("向Tracker注册失败: %v", err), true)
		job.updateStatus(JobStatusFailed, "网络连接失败")
		return
	}

	// 开始做种
	job.updateStatus(JobStatusSeeding, "开始做种")

	if err := e.seedLoop(job); err != nil {
		if job.ctx.Err() == nil { // 不是取消导致的错误
			job.sendError(ErrorTypeNetwork, fmt.Sprintf("做种过程中出错: %v", err), true)
			job.updateStatus(JobStatusFailed, "做种过程中出错")
		}
		return
	}
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
			Message:   "任务已创建",
			Timestamp: time.Now(),
		},
	}
}

// registerJob 注册任务到引擎
func (e *Engine) registerJob(job *Job) {
	e.jobsMu.Lock()
	e.jobs[job.id] = job
	e.jobsMu.Unlock()
}

// getAllJobs 获取所有任务的快照
func (e *Engine) getAllJobs() []*Job {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()

	jobs := make([]*Job, 0, len(e.jobs))
	for _, job := range e.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// cleanupJob 清理完成的任务
func (e *Engine) cleanupJob(job *Job) {
	close(job.doneCh)
	// 可选：自动移除已完成的任务
	// e.removeJob(job.id)
}

// 网络协议相关方法

// parseMetaFile 解析元数据文件
func (e *Engine) parseMetaFile(job *Job) error {
	job.sendProgress("正在解析元数据文件...")
	// 模拟解析过程
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// connectToTracker 连接到Tracker服务器（保留用于其他地方调用）
func (e *Engine) connectToTracker(job *Job) error {
	job.sendProgress("正在连接Tracker...")
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(500 * time.Millisecond):
		job.sendProgress("已连接到Tracker，发现2个Peer")
		return nil
	}
}

// connectToNetwork 连接到网络（替代原来分离的方法）
func (e *Engine) connectToNetwork(job *Job) error {
	// 1. 连接到Tracker
	job.sendProgress("正在连接Tracker...")
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(500 * time.Millisecond):
	}

	// 2. 连接到Peer
	peers := []string{"192.168.1.100:6881", "192.168.1.101:6881"}

	job.updateProgress(func(p *ProgressUpdate) {
		p.ConnectedPeers = len(peers)
		p.Message = fmt.Sprintf("已连接到 %d 个Peer", len(peers))
	})

	for _, peer := range peers {
		select {
		case <-job.ctx.Done():
			return job.ctx.Err()
		default:
		}

		job.sendProgress(fmt.Sprintf("连接到Peer: %s", peer))
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// downloadLoop 主要的下载循环
func (e *Engine) downloadLoop(job *Job) error {
	const totalPieces = 100
	const pieceSize = 1024 * 1024 // 1MB per piece

	// 初始化进度信息
	job.updateProgress(func(p *ProgressUpdate) {
		p.TotalSize = totalPieces * pieceSize
		p.TotalPieces = totalPieces
		p.CompletedPieces = 0
		p.StartTime = time.Now()
		p.Message = "开始下载"
	})

	for i := 0; i < totalPieces; i++ {
		select {
		case <-job.ctx.Done():
			return job.ctx.Err()
		default:
		}

		// 模拟下载一个分片
		time.Sleep(50 * time.Millisecond)

		// 更新详细进度信息
		downloadedSize := int64((i + 1) * pieceSize)
		elapsedTime := time.Since(time.Now().Add(-time.Duration(i+1) * 50 * time.Millisecond))

		job.updateProgress(func(p *ProgressUpdate) {
			p.DownloadedSize = downloadedSize
			p.CompletedPieces = i + 1
			p.PendingPieces = totalPieces - i - 1
			p.ElapsedTime = elapsedTime

			// 计算下载速度
			if elapsedTime.Seconds() > 0 {
				p.DownloadSpeed = int64(float64(downloadedSize) / elapsedTime.Seconds())
				p.AverageSpeed = p.DownloadSpeed
			}

			// 估算剩余时间
			if p.DownloadSpeed > 0 {
				remainingBytes := p.TotalSize - downloadedSize
				p.EstimatedTime = time.Duration(float64(remainingBytes)/float64(p.DownloadSpeed)) * time.Second
			}

			p.Message = fmt.Sprintf("下载中 %.1f%% (%s/%s)",
				p.PercentComplete(), FormatBytes(downloadedSize), FormatBytes(p.TotalSize))
		})

		// 定期发送进度消息
		if i%10 == 0 || i == totalPieces-1 {
			job.sendProgress(fmt.Sprintf("已下载 %d/%d 分片 (%s)",
				i+1, totalPieces, FormatBytes(downloadedSize)))
		}
	}

	return nil
}

// verifyFile 验证下载的文件完整性
func (e *Engine) verifyFile(job *Job) error {
	job.sendProgress("正在验证文件完整性...")
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(1 * time.Second):
		return nil
	}
}

// createMetaFile 创建元数据文件（用于做种）
func (e *Engine) createMetaFile(job *Job) error {
	job.sendProgress("正在分析文件并创建元数据...")
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(2 * time.Second):
		return nil
	}
}

// verifyLocalFile 验证本地文件（做种前）
func (e *Engine) verifyLocalFile(job *Job) error {
	job.sendProgress("正在验证本地文件...")
	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(1 * time.Second):
		return nil
	}
}

// announceToTracker 向Tracker宣告
func (e *Engine) announceToTracker(job *Job, seeding bool) error {
	if seeding {
		job.sendProgress("向Tracker注册为种子...")
	} else {
		job.sendProgress("向Tracker宣告下载...")
	}

	select {
	case <-job.ctx.Done():
		return job.ctx.Err()
	case <-time.After(300 * time.Millisecond):
		return nil
	}
}

// seedLoop 做种循环
func (e *Engine) seedLoop(job *Job) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	uploadedSize := int64(0)
	startTime := time.Now()

	// 初始化做种状态
	job.updateProgress(func(p *ProgressUpdate) {
		p.TotalSize = 100 * 1024 * 1024 // 假设100MB文件
		p.DownloadedSize = p.TotalSize  // 做种时已完全下载
		p.UploadedSize = 0
		p.StartTime = startTime
		p.ConnectedPeers = 2 // 模拟连接的peer数
		p.Message = "做种中，等待连接..."
	})

	for {
		select {
		case <-job.ctx.Done():
			return nil // 正常退出
		case <-ticker.C:
			// 模拟上传数据
			uploadedSize += 1024 * 1024 // 假设每10秒上传1MB
			elapsedTime := time.Since(startTime)

			job.updateProgress(func(p *ProgressUpdate) {
				p.UploadedSize = uploadedSize
				p.ElapsedTime = elapsedTime

				// 计算上传速度和分享率
				if elapsedTime.Seconds() > 0 {
					p.UploadSpeed = int64(float64(uploadedSize) / elapsedTime.Seconds())
				}
				if p.DownloadedSize > 0 {
					p.Ratio = float64(uploadedSize) / float64(p.DownloadedSize)
				}

				// 更新peer信息
				p.ConnectedPeers = 2 + (int(uploadedSize/(1024*1024)) % 3) // 模拟peer数变化
				p.ActivePeers = min(p.ConnectedPeers, 2)

				p.Message = fmt.Sprintf("做种中，已上传: %s (分享率: %.2f)",
					FormatBytes(uploadedSize), p.Ratio)
			})

			job.sendProgress(fmt.Sprintf("做种中，已上传: %s", FormatBytes(uploadedSize)))

			// 更新引擎统计
			e.updateUploadStats(1024 * 1024)
		}
	}
}

// 工具函数

// generateJobID 生成唯一的任务ID
func generateJobID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// 私有方法

// initNetwork 初始化网络组件
func (e *Engine) initNetwork() error {
	// 实际实现会在这里初始化网络监听器、连接池等
	return nil
}

// backgroundTasks 后台任务处理
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

// updateStats 更新统计信息
func (e *Engine) updateStats() {
	e.jobsMu.RLock()
	activeJobs := len(e.jobs)
	e.jobsMu.RUnlock()

	e.statsMu.Lock()
	e.stats.ActiveJobs = activeJobs
	e.statsMu.Unlock()
}

// updateUploadStats 更新上传统计
func (e *Engine) updateUploadStats(bytes int64) {
	e.statsMu.Lock()
	e.stats.TotalUploaded += bytes
	e.statsMu.Unlock()
}

// removeJob 移除已完成的任务
func (e *Engine) removeJob(jobID string) {
	e.jobsMu.Lock()
	delete(e.jobs, jobID)
	e.jobsMu.Unlock()
}
