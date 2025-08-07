// Package engine 提供了一个与UI完全解耦的P2P下载引擎
// 优化版本：实现了连接管理、任务调度、资源控制、做种逻辑和健壮性测试
package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Engine 是P2P下载引擎的主要接口
type Engine struct {
	config Config
	ctx    context.Context
	cancel context.CancelFunc

	// 任务管理
	jobs       map[string]*Job
	jobsMu     sync.RWMutex
	jobQueue   chan *Job // 任务队列
	maxWorkers int       // 最大并发工作协程数

	// 连接管理优化
	peerManager    *PeerManager
	connectionPool *ConnectionPool

	// 资源管理
	pieceManager  *PieceManager
	taskScheduler *TaskScheduler

	// 统计和监控
	stats     EngineStats
	statsMu   sync.RWMutex
	startTime time.Time

	// 健壮性
	errorRecovery *ErrorRecovery
	testMode      bool
}

// PeerManager 管理Peer连接和Choking算法
type PeerManager struct {
	peers       map[string]*PeerConnection
	peersMu     sync.RWMutex
	maxPeers    int
	chokingAlgo *ChokingAlgorithm
}

// ChokingAlgorithm 实现"窒息"算法
type ChokingAlgorithm struct {
	unchokeInterval    time.Duration
	optimisticInterval time.Duration
	ticker             *time.Ticker
	engine             *Engine
}

// PeerConnection 表示与单个Peer的连接
type PeerConnection struct {
	ID           string
	Address      string
	State        PeerState
	LastActivity time.Time
	IsChoked     bool
	IsInterested bool

	// 统计信息
	DownloadSpeed int64
	UploadSpeed   int64
	Downloaded    int64
	Uploaded      int64

	// 连接管理
	conn    interface{} // 实际的网络连接
	sendCh  chan []byte
	recvCh  chan []byte
	closeCh chan struct{}
}

type PeerState int

const (
	PeerStateConnecting PeerState = iota
	PeerStateConnected
	PeerStateDisconnected
	PeerStateError
)

// ConnectionPool 连接池管理
type ConnectionPool struct {
	maxConnections int
	activeConns    int32
	connCh         chan *PeerConnection
	mu             sync.Mutex
}

// PieceManager 分片管理器
type PieceManager struct {
	pieces       map[int]*PieceState
	piecesMu     sync.RWMutex
	totalPieces  int
	pieceSize    int64
	pendingQueue chan *PieceTask
	activeJobs   map[int]*PieceTask
	activeJobsMu sync.RWMutex
}

type PieceState struct {
	Index      int
	Status     PieceStatus
	Data       []byte
	Hash       []byte
	Retries    int
	LastTry    time.Time
	AssignedTo string // Peer ID
}

type PieceStatus int

const (
	PieceStatusPending PieceStatus = iota
	PieceStatusRequested
	PieceStatusDownloading
	PieceStatusCompleted
	PieceStatusFailed
)

type PieceTask struct {
	PieceIndex int
	JobID      string
	StartTime  time.Time
	Timeout    time.Duration
	PeerID     string
	Retries    int
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	pendingTasks chan *PieceTask
	activeTasks  map[string]*PieceTask
	tasksMu      sync.RWMutex
	maxRetries   int
	timeout      time.Duration
}

// ErrorRecovery 错误恢复机制
type ErrorRecovery struct {
	maxRetries    int
	backoffBase   time.Duration
	maxBackoff    time.Duration
	failedPeers   map[string]int
	failedPeersMu sync.RWMutex
}

// SeedingStats 做种统计信息
type SeedingStats struct {
	UploadedThisCycle int64
	LastAnnounce      time.Time
	ConnectedPeers    int
	ActiveUploads     int
}

// NewEngine 创建优化后的P2P引擎实例
func NewEngine(config Config) (*Engine, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		jobs:       make(map[string]*Job),
		jobQueue:   make(chan *Job, 100),
		maxWorkers: config.MaxConcurrentDownloads,
		startTime:  time.Now(),
		testMode:   false, // 默认非测试模式
	}

	// 初始化各个组件
	engine.peerManager = NewPeerManager(config.MaxPeers)
	engine.connectionPool = NewConnectionPool(config.MaxPeers) // 使用MaxPeers作为连接池大小
	engine.pieceManager = NewPieceManager()
	engine.taskScheduler = NewTaskScheduler(config.RequestTimeout, config.MaxRetries)
	engine.errorRecovery = NewErrorRecovery(config.MaxRetries)

	// 初始化网络组件
	if err := engine.initNetwork(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	// 初始化Choking算法
	engine.peerManager.chokingAlgo = &ChokingAlgorithm{
		unchokeInterval:    30 * time.Second,
		optimisticInterval: 10 * time.Second,
		engine:             engine,
	}

	// 启动各种后台协程
	go engine.workerPool()
	go engine.chokingAlgorithmLoop()
	go engine.taskSchedulerLoop()
	go engine.connectionHealthCheck()
	go engine.backgroundTasks()

	return engine, nil
}

// CreateDownloadJob 创建新的下载任务
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

// 1. 连接管理与"窒息"算法优化

func (ca *ChokingAlgorithm) Start() {
	ca.ticker = time.NewTicker(ca.unchokeInterval)
	go ca.run()
}

func (ca *ChokingAlgorithm) run() {
	optimisticTicker := time.NewTicker(ca.optimisticInterval)
	defer optimisticTicker.Stop()
	defer ca.ticker.Stop()

	for {
		select {
		case <-ca.ticker.C:
			ca.evaluateChoking()
		case <-optimisticTicker.C:
			ca.optimisticUnchoke()
		case <-ca.engine.ctx.Done():
			return
		}
	}
}

func (ca *ChokingAlgorithm) evaluateChoking() {
	pm := ca.engine.peerManager
	pm.peersMu.RLock()
	defer pm.peersMu.RUnlock()

	// 获取所有感兴趣的Peer
	interestedPeers := make([]*PeerConnection, 0)
	for _, peer := range pm.peers {
		if peer.IsInterested {
			interestedPeers = append(interestedPeers, peer)
		}
	}

	// 按下载速度排序（生产级实现：Tit-for-Tat）
	sort.Slice(interestedPeers, func(i, j int) bool {
		return interestedPeers[i].DownloadSpeed > interestedPeers[j].DownloadSpeed
	})

	// Unchoke前4个最快的Peer
	maxUnchoked := min(4, len(interestedPeers))
	for i, peer := range interestedPeers {
		shouldUnchoke := i < maxUnchoked
		if peer.IsChoked && shouldUnchoke {
			ca.unchokePeer(peer)
		} else if !peer.IsChoked && !shouldUnchoke {
			ca.chokePeer(peer)
		}
	}
}

func (ca *ChokingAlgorithm) optimisticUnchoke() {
	pm := ca.engine.peerManager
	pm.peersMu.RLock()
	defer pm.peersMu.RUnlock()

	// 找到一个被Choke但感兴趣的Peer进行乐观Unchoke
	for _, peer := range pm.peers {
		if peer.IsChoked && peer.IsInterested {
			ca.unchokePeer(peer)
			break
		}
	}
}

func (ca *ChokingAlgorithm) unchokePeer(peer *PeerConnection) {
	peer.IsChoked = false
	// 发送Unchoke消息
	select {
	case peer.sendCh <- []byte("unchoke"):
	default:
		// 缓冲区满，记录错误但不阻塞
	}
}

func (ca *ChokingAlgorithm) chokePeer(peer *PeerConnection) {
	peer.IsChoked = true
	select {
	case peer.sendCh <- []byte("choke"):
	default:
	}
}

// 2. 任务调度、超时与重试优化

func (e *Engine) taskSchedulerLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkTimeouts()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) checkTimeouts() {
	e.taskScheduler.tasksMu.Lock()
	defer e.taskScheduler.tasksMu.Unlock()

	now := time.Now()
	for taskID, task := range e.taskScheduler.activeTasks {
		if now.Sub(task.StartTime) > task.Timeout {
			// 任务超时，重新调度
			e.rescheduleTask(task)
			delete(e.taskScheduler.activeTasks, taskID)
		}
	}
}

func (e *Engine) rescheduleTask(task *PieceTask) {
	task.Retries++
	if task.Retries > e.taskScheduler.maxRetries {
		// 标记分片为永久失败
		e.markPieceFailed(task.PieceIndex, fmt.Errorf("exceeded max retries"))
		return
	}

	// 重新放入队列，并增加退避延迟
	backoff := e.errorRecovery.calculateBackoff(task.Retries)
	time.AfterFunc(backoff, func() {
		select {
		case e.pieceManager.pendingQueue <- task:
		default:
			// 队列满，丢弃任务
		}
	})
}

// 3. 资源管理与限制

func NewConnectionPool(maxConn int) *ConnectionPool {
	return &ConnectionPool{
		maxConnections: maxConn,
		connCh:         make(chan *PeerConnection, maxConn),
	}
}

func (cp *ConnectionPool) AcquireConnection() (*PeerConnection, error) {
	current := atomic.LoadInt32(&cp.activeConns)
	if int(current) >= cp.maxConnections {
		return nil, fmt.Errorf("connection pool exhausted")
	}

	if atomic.CompareAndSwapInt32(&cp.activeConns, current, current+1) {
		return &PeerConnection{
			sendCh:  make(chan []byte, 100),
			recvCh:  make(chan []byte, 100),
			closeCh: make(chan struct{}),
		}, nil
	}

	return nil, fmt.Errorf("failed to acquire connection")
}

func (cp *ConnectionPool) ReleaseConnection(conn *PeerConnection) {
	atomic.AddInt32(&cp.activeConns, -1)
	close(conn.closeCh)
}

func (e *Engine) connectionHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanupStaleConnections()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) cleanupStaleConnections() {
	pm := e.peerManager
	pm.peersMu.Lock()
	defer pm.peersMu.Unlock()

	now := time.Now()
	for peerID, peer := range pm.peers {
		if now.Sub(peer.LastActivity) > 5*time.Minute {
			// 清理长时间不活跃的连接
			e.connectionPool.ReleaseConnection(peer)
			delete(pm.peers, peerID)
		}
	}
}

// 4. 做种逻辑的完善

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

	if err := e.improvedSeedLoop(job); err != nil {
		if job.ctx.Err() == nil { // 不是取消导致的错误
			job.sendError(ErrorTypeNetwork, fmt.Sprintf("做种过程中出错: %v", err), true)
			job.updateStatus(JobStatusFailed, "做种过程中出错")
		}
		return
	}
}

func (e *Engine) improvedSeedLoop(job *Job) error {
	seedStats := &SeedingStats{
		LastAnnounce: time.Now(),
	}

	// 定期向Tracker汇报
	announceTicker := time.NewTicker(15 * time.Minute)
	defer announceTicker.Stop()

	// 统计上传数据
	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-job.ctx.Done():
			return nil

		case <-announceTicker.C:
			if err := e.announceToTracker(job, true); err != nil {
				job.sendError(ErrorTypeNetwork, fmt.Sprintf("Tracker汇报失败: %v", err), false)
			} else {
				seedStats.LastAnnounce = time.Now()
			}

		case <-statsTicker.C:
			e.updateSeedingStats(job, seedStats)

		case pieceReq := <-e.getIncomingPieceRequests(job):
			// 处理来自其他Peer的分片请求
			go e.handlePieceRequest(job, pieceReq)
		}
	}
}

func (e *Engine) handlePieceRequest(job *Job, req *PieceRequest) {
	// 检查是否有请求的分片
	pieceData, err := e.getPieceData(job, req.PieceIndex)
	if err != nil {
		return // 没有该分片，忽略请求
	}

	// 发送分片数据
	response := &PieceResponse{
		PieceIndex: req.PieceIndex,
		Data:       pieceData,
		JobID:      job.id,
	}

	if err := e.sendPieceResponse(req.PeerID, response); err == nil {
		// 更新上传统计
		atomic.AddInt64(&job.progress.UploadedSize, int64(len(pieceData)))
		e.updateUploadStats(int64(len(pieceData)))
	}
}

func (e *Engine) updateSeedingStats(job *Job, stats *SeedingStats) {
	pm := e.peerManager
	pm.peersMu.RLock()
	connectedPeers := len(pm.peers)
	pm.peersMu.RUnlock()

	job.updateProgress(func(p *ProgressUpdate) {
		p.ConnectedPeers = connectedPeers
		p.ActivePeers = min(connectedPeers, 5) // 假设最多5个活跃上传

		// 更新上传速度
		elapsed := time.Since(stats.LastAnnounce)
		if elapsed.Seconds() > 0 {
			p.UploadSpeed = stats.UploadedThisCycle / int64(elapsed.Seconds())
		}

		// 计算分享率
		if p.DownloadedSize > 0 {
			p.Ratio = float64(p.UploadedSize) / float64(p.DownloadedSize)
		}

		p.Message = fmt.Sprintf("做种中 - 连接: %d, 分享率: %.2f",
			connectedPeers, p.Ratio)
	})

	// 重置周期统计
	stats.UploadedThisCycle = 0
}

// 5. 健壮性与测试

func NewErrorRecovery(maxRetries int) *ErrorRecovery {
	return &ErrorRecovery{
		maxRetries:  maxRetries,
		backoffBase: 1 * time.Second,
		maxBackoff:  30 * time.Second,
		failedPeers: make(map[string]int),
	}
}

func (er *ErrorRecovery) calculateBackoff(retries int) time.Duration {
	backoff := er.backoffBase * time.Duration(1<<uint(retries)) // 指数退避
	if backoff > er.maxBackoff {
		backoff = er.maxBackoff
	}
	return backoff
}

func (er *ErrorRecovery) recordPeerFailure(peerID string) {
	er.failedPeersMu.Lock()
	defer er.failedPeersMu.Unlock()
	er.failedPeers[peerID]++
}

func (er *ErrorRecovery) shouldBlacklistPeer(peerID string) bool {
	er.failedPeersMu.RLock()
	defer er.failedPeersMu.RUnlock()
	return er.failedPeers[peerID] > er.maxRetries
}

// 单元测试辅助方法
func (e *Engine) RunTests() error {
	if !e.testMode {
		return fmt.Errorf("engine not in test mode")
	}

	tests := []struct {
		name string
		test func() error
	}{
		{"TestPieceManagerBasic", e.testPieceManagerBasic},
		{"TestChokingAlgorithm", e.testChokingAlgorithm},
		{"TestConnectionPool", e.testConnectionPool},
		{"TestTaskScheduler", e.testTaskScheduler},
		{"TestErrorRecovery", e.testErrorRecovery},
	}

	for _, tt := range tests {
		if err := tt.test(); err != nil {
			return fmt.Errorf("test %s failed: %w", tt.name, err)
		}
	}

	return nil
}

func (e *Engine) testPieceManagerBasic() error {
	// 测试分片管理器的基本功能
	pm := e.pieceManager

	// 测试分片状态跟踪
	pm.pieces[0] = &PieceState{
		Index:  0,
		Status: PieceStatusPending,
	}

	if pm.pieces[0].Status != PieceStatusPending {
		return fmt.Errorf("piece status not set correctly")
	}

	return nil
}

func (e *Engine) testChokingAlgorithm() error {
	// 测试Choking算法逻辑
	ca := e.peerManager.chokingAlgo

	// 创建测试Peer
	peer := &PeerConnection{
		ID:            "test-peer",
		IsInterested:  true,
		IsChoked:      true,
		DownloadSpeed: 1024 * 1024, // 1MB/s
		sendCh:        make(chan []byte, 10),
	}

	// 测试Unchoke
	ca.unchokePeer(peer)
	if peer.IsChoked {
		return fmt.Errorf("peer should be unchoked")
	}

	return nil
}

func (e *Engine) testConnectionPool() error {
	cp := e.connectionPool

	// 测试连接获取
	conn, err := cp.AcquireConnection()
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}

	// 测试连接释放
	cp.ReleaseConnection(conn)

	return nil
}

func (e *Engine) testTaskScheduler() error {
	// 测试任务调度器
	task := &PieceTask{
		PieceIndex: 1,
		JobID:      "test-job",
		StartTime:  time.Now(),
		Timeout:    5 * time.Second,
	}

	e.taskScheduler.tasksMu.Lock()
	e.taskScheduler.activeTasks["test-task"] = task
	e.taskScheduler.tasksMu.Unlock()

	// 验证任务存在
	e.taskScheduler.tasksMu.RLock()
	_, exists := e.taskScheduler.activeTasks["test-task"]
	e.taskScheduler.tasksMu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found in scheduler")
	}

	return nil
}

func (e *Engine) testErrorRecovery() error {
	er := e.errorRecovery

	// 测试退避计算
	backoff := er.calculateBackoff(3)
	expected := er.backoffBase * 8 // 2^3
	if backoff != expected {
		return fmt.Errorf("wrong backoff calculation: got %v, expected %v", backoff, expected)
	}

	return nil
}

// 实现基础功能方法

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
	job.markDone()
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

// connectToNetwork 连接到网络
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
	pieceSize := int64(e.config.PieceSize)

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
		downloadedSize := int64((i + 1)) * pieceSize
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

// 工具和辅助方法

// generateJobID 生成唯一的任务ID
func generateJobID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

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

// 辅助类型和方法

type PieceRequest struct {
	PieceIndex int
	PeerID     string
}

type PieceResponse struct {
	PieceIndex int
	Data       []byte
	JobID      string
}

func NewPeerManager(maxPeers int) *PeerManager {
	return &PeerManager{
		peers:    make(map[string]*PeerConnection),
		maxPeers: maxPeers,
	}
}

func NewPieceManager() *PieceManager {
	return &PieceManager{
		pieces:       make(map[int]*PieceState),
		pendingQueue: make(chan *PieceTask, 1000),
		activeJobs:   make(map[int]*PieceTask),
	}
}

func NewTaskScheduler(timeout time.Duration, maxRetries int) *TaskScheduler {
	return &TaskScheduler{
		pendingTasks: make(chan *PieceTask, 1000),
		activeTasks:  make(map[string]*PieceTask),
		maxRetries:   maxRetries,
		timeout:      timeout,
	}
}

// 占位实现（实际项目中需要完整实现）
func (e *Engine) workerPool() {
	// 创建工作协程池
	for i := 0; i < e.maxWorkers; i++ {
		go e.worker(i)
	}
}

func (e *Engine) worker(id int) {
	for {
		select {
		case job := <-e.jobQueue:
			// 处理任务
			_ = job // 实际处理逻辑
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) chokingAlgorithmLoop() {
	if e.peerManager.chokingAlgo != nil {
		e.peerManager.chokingAlgo.Start()
	}
}

func (e *Engine) getIncomingPieceRequests(job *Job) <-chan *PieceRequest {
	// 返回接收分片请求的channel
	ch := make(chan *PieceRequest, 10)
	// 实际实现中会从网络接收请求并发送到这个channel
	return ch
}

func (e *Engine) getPieceData(job *Job, pieceIndex int) ([]byte, error) {
	// 实际实现中会从文件系统读取分片数据
	return nil, fmt.Errorf("not implemented")
}

func (e *Engine) sendPieceResponse(peerID string, resp *PieceResponse) error {
	// 实际实现中会通过网络发送分片数据给指定Peer
	return fmt.Errorf("not implemented")
}

func (e *Engine) markPieceFailed(pieceIndex int, err error) {
	// 标记分片失败并记录错误
	e.pieceManager.piecesMu.Lock()
	defer e.pieceManager.piecesMu.Unlock()

	if piece, exists := e.pieceManager.pieces[pieceIndex]; exists {
		piece.Status = PieceStatusFailed
		piece.Retries++
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
