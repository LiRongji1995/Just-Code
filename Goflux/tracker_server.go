package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// 配置结构
type Config struct {
	ServerPort     int    `json:"server_port"`
	RedisAddr      string `json:"redis_addr"`
	RedisPassword  string `json:"redis_password"`
	RedisDB        int    `json:"redis_db"`
	CleanInterval  int    `json:"clean_interval"` // 清理间隔(秒)
	PeerTTL        int    `json:"peer_ttl"`       // Peer TTL(秒)
	MaxPeersReturn int    `json:"max_peers_return"`
}

// Peer信息结构
type PeerInfo struct {
	PeerID     string    `json:"peer_id"`
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	Uploaded   int64     `json:"uploaded"`
	Downloaded int64     `json:"downloaded"`
	Left       int64     `json:"left"`
	Event      string    `json:"event,omitempty"` // started, stopped, completed
	LastSeen   time.Time `json:"last_seen"`
}

// Announce请求结构
type AnnounceRequest struct {
	FileHash   string `json:"file_hash" binding:"required"`
	PeerID     string `json:"peer_id" binding:"required"`
	IP         string `json:"ip"`
	Port       int    `json:"port" binding:"required"`
	Uploaded   int64  `json:"uploaded"`
	Downloaded int64  `json:"downloaded"`
	Left       int64  `json:"left"`
	Event      string `json:"event"`
	NumWant    int    `json:"numwant"`
}

// Announce响应结构
type AnnounceResponse struct {
	Interval  int        `json:"interval"`
	Peers     []PeerInfo `json:"peers"`
	Seeders   int        `json:"seeders"`
	Leechers  int        `json:"leechers"`
	Timestamp int64      `json:"timestamp"`
}

// Scrape响应结构
type ScrapeResponse struct {
	FileHash  string `json:"file_hash"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Completed int    `json:"completed"`
	Timestamp int64  `json:"timestamp"`
}

// 统计信息
type TrackerStats struct {
	TotalFiles      int `json:"total_files"`
	TotalPeers      int `json:"total_peers"`
	TotalSeeders    int `json:"total_seeders"`
	TotalLeechers   int `json:"total_leechers"`
	RequestsPerHour int `json:"requests_per_hour"`
}

// Tracker服务器
type TrackerServer struct {
	config  *Config
	redis   *redis.Client
	logger  *zap.Logger
	metrics *TrackerMetrics
	ctx     context.Context
	cancel  context.CancelFunc
}

// Prometheus指标
type TrackerMetrics struct {
	announceRequests prometheus.Counter
	scrapeRequests   prometheus.Counter
	activePeers      prometheus.Gauge
	activeFiles      prometheus.Gauge
	errorCounter     prometheus.Counter
}

// 初始化指标
func NewTrackerMetrics() *TrackerMetrics {
	return &TrackerMetrics{
		announceRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tracker_announce_requests_total",
			Help: "Total number of announce requests",
		}),
		scrapeRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tracker_scrape_requests_total",
			Help: "Total number of scrape requests",
		}),
		activePeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tracker_active_peers",
			Help: "Number of active peers",
		}),
		activeFiles: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tracker_active_files",
			Help: "Number of active files",
		}),
		errorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tracker_errors_total",
			Help: "Total number of errors",
		}),
	}
}

// 注册指标
func (m *TrackerMetrics) Register() {
	prometheus.MustRegister(
		m.announceRequests,
		m.scrapeRequests,
		m.activePeers,
		m.activeFiles,
		m.errorCounter,
	)
}

// 创建新的Tracker服务器
func NewTrackerServer(config *Config) (*TrackerServer, error) {
	// 初始化logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

	// 初始化Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// 测试Redis连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	// 初始化指标
	metrics := NewTrackerMetrics()
	metrics.Register()

	ctx, cancel := context.WithCancel(context.Background())

	server := &TrackerServer{
		config:  config,
		redis:   rdb,
		logger:  logger,
		metrics: metrics,
		ctx:     ctx,
		cancel:  cancel,
	}

	return server, nil
}

// Redis键生成器
func (ts *TrackerServer) getPeerKey(fileHash, peerID string) string {
	return fmt.Sprintf("file:%s:peer:%s", fileHash, peerID)
}

func (ts *TrackerServer) getFileKey(fileHash string) string {
	return fmt.Sprintf("file:%s:peers", fileHash)
}

func (ts *TrackerServer) getStatsKey(fileHash string) string {
	return fmt.Sprintf("file:%s:stats", fileHash)
}

// 处理announce请求
func (ts *TrackerServer) handleAnnounce(c *gin.Context) {
	ts.metrics.announceRequests.Inc()

	var req AnnounceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ts.metrics.errorCounter.Inc()
		ts.logger.Error("Invalid announce request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 获取客户端IP（如果请求中没有提供）
	if req.IP == "" {
		req.IP = c.ClientIP()
	}

	// 设置默认numwant
	if req.NumWant == 0 {
		req.NumWant = ts.config.MaxPeersReturn
	}

	// 创建peer信息
	peer := PeerInfo{
		PeerID:     req.PeerID,
		IP:         req.IP,
		Port:       req.Port,
		Uploaded:   req.Uploaded,
		Downloaded: req.Downloaded,
		Left:       req.Left,
		Event:      req.Event,
		LastSeen:   time.Now(),
	}

	ctx := context.Background()

	// 处理不同事件
	switch req.Event {
	case "stopped":
		// 移除peer
		ts.removePeer(ctx, req.FileHash, req.PeerID)
	default:
		// 添加或更新peer
		ts.addOrUpdatePeer(ctx, req.FileHash, peer)
	}

	// 获取peer列表
	peers, err := ts.getPeers(ctx, req.FileHash, req.PeerID, req.NumWant)
	if err != nil {
		ts.metrics.errorCounter.Inc()
		ts.logger.Error("Failed to get peers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 获取统计信息
	stats, err := ts.getFileStats(ctx, req.FileHash)
	if err != nil {
		ts.logger.Warn("Failed to get file stats", zap.Error(err))
		stats = &ScrapeResponse{}
	}

	response := AnnounceResponse{
		Interval:  1800, // 30分钟
		Peers:     peers,
		Seeders:   stats.Seeders,
		Leechers:  stats.Leechers,
		Timestamp: time.Now().Unix(),
	}

	c.JSON(http.StatusOK, response)
}

// 添加或更新peer
func (ts *TrackerServer) addOrUpdatePeer(ctx context.Context, fileHash string, peer PeerInfo) error {
	// 序列化peer信息
	peerData, err := json.Marshal(peer)
	if err != nil {
		return err
	}

	// 存储peer信息
	peerKey := ts.getPeerKey(fileHash, peer.PeerID)
	if err := ts.redis.Set(ctx, peerKey, peerData, time.Duration(ts.config.PeerTTL)*time.Second).Err(); err != nil {
		return err
	}

	// 添加到文件的peer集合
	fileKey := ts.getFileKey(fileHash)
	if err := ts.redis.SAdd(ctx, fileKey, peer.PeerID).Err(); err != nil {
		return err
	}

	// 设置文件key的TTL
	ts.redis.Expire(ctx, fileKey, time.Duration(ts.config.PeerTTL)*time.Second)

	// 更新统计信息
	ts.updateFileStats(ctx, fileHash, peer)

	return nil
}

// 移除peer
func (ts *TrackerServer) removePeer(ctx context.Context, fileHash, peerID string) error {
	// 删除peer信息
	peerKey := ts.getPeerKey(fileHash, peerID)
	ts.redis.Del(ctx, peerKey)

	// 从文件的peer集合中移除
	fileKey := ts.getFileKey(fileHash)
	ts.redis.SRem(ctx, fileKey, peerID)

	return nil
}

// 获取peer列表
func (ts *TrackerServer) getPeers(ctx context.Context, fileHash, excludePeerID string, numWant int) ([]PeerInfo, error) {
	fileKey := ts.getFileKey(fileHash)

	// 获取所有peer ID
	peerIDs, err := ts.redis.SMembers(ctx, fileKey).Result()
	if err != nil {
		return nil, err
	}

	var peers []PeerInfo
	for _, peerID := range peerIDs {
		if peerID == excludePeerID {
			continue // 不返回请求者自己
		}

		peerKey := ts.getPeerKey(fileHash, peerID)
		peerData, err := ts.redis.Get(ctx, peerKey).Result()
		if err != nil {
			// peer可能已过期，从集合中清理
			ts.redis.SRem(ctx, fileKey, peerID)
			continue
		}

		var peer PeerInfo
		if err := json.Unmarshal([]byte(peerData), &peer); err != nil {
			continue
		}

		peers = append(peers, peer)

		if len(peers) >= numWant {
			break
		}
	}

	return peers, nil
}

// 处理scrape请求
func (ts *TrackerServer) handleScrape(c *gin.Context) {
	ts.metrics.scrapeRequests.Inc()

	fileHash := c.Query("file_hash")
	if fileHash == "" {
		ts.metrics.errorCounter.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_hash parameter required"})
		return
	}

	stats, err := ts.getFileStats(context.Background(), fileHash)
	if err != nil {
		ts.metrics.errorCounter.Inc()
		ts.logger.Error("Failed to get file stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// 获取文件统计信息
func (ts *TrackerServer) getFileStats(ctx context.Context, fileHash string) (*ScrapeResponse, error) {
	fileKey := ts.getFileKey(fileHash)

	// 获取所有peer
	peerIDs, err := ts.redis.SMembers(ctx, fileKey).Result()
	if err != nil {
		return nil, err
	}

	var seeders, leechers int
	for _, peerID := range peerIDs {
		peerKey := ts.getPeerKey(fileHash, peerID)
		peerData, err := ts.redis.Get(ctx, peerKey).Result()
		if err != nil {
			// 清理过期peer
			ts.redis.SRem(ctx, fileKey, peerID)
			continue
		}

		var peer PeerInfo
		if err := json.Unmarshal([]byte(peerData), &peer); err != nil {
			continue
		}

		if peer.Left == 0 {
			seeders++
		} else {
			leechers++
		}
	}

	// 从Redis获取completed计数
	completedKey := fmt.Sprintf("file:%s:completed", fileHash)
	completed, _ := ts.redis.Get(ctx, completedKey).Int()

	return &ScrapeResponse{
		FileHash:  fileHash,
		Seeders:   seeders,
		Leechers:  leechers,
		Completed: completed,
		Timestamp: time.Now().Unix(),
	}, nil
}

// 更新文件统计信息
func (ts *TrackerServer) updateFileStats(ctx context.Context, fileHash string, peer PeerInfo) {
	// 如果是completed事件，增加完成计数
	if peer.Event == "completed" {
		completedKey := fmt.Sprintf("file:%s:completed", fileHash)
		ts.redis.Incr(ctx, completedKey)
		ts.redis.Expire(ctx, completedKey, 24*time.Hour) // 24小时过期
	}
}

// 获取整体统计信息
func (ts *TrackerServer) handleStats(c *gin.Context) {
	ctx := context.Background()

	// 获取所有文件key
	fileKeys, err := ts.redis.Keys(ctx, "file:*:peers").Result()
	if err != nil {
		ts.metrics.errorCounter.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	totalFiles := len(fileKeys)
	totalPeers := 0
	totalSeeders := 0
	totalLeechers := 0

	for _, fileKey := range fileKeys {
		peerIDs, err := ts.redis.SMembers(ctx, fileKey).Result()
		if err != nil {
			continue
		}

		totalPeers += len(peerIDs)

		// 计算seeder和leecher
		fileHash := strings.Split(fileKey, ":")[1]
		for _, peerID := range peerIDs {
			peerKey := ts.getPeerKey(fileHash, peerID)
			peerData, err := ts.redis.Get(ctx, peerKey).Result()
			if err != nil {
				continue
			}

			var peer PeerInfo
			if err := json.Unmarshal([]byte(peerData), &peer); err != nil {
				continue
			}

			if peer.Left == 0 {
				totalSeeders++
			} else {
				totalLeechers++
			}
		}
	}

	// 更新Prometheus指标
	ts.metrics.activePeers.Set(float64(totalPeers))
	ts.metrics.activeFiles.Set(float64(totalFiles))

	stats := TrackerStats{
		TotalFiles:    totalFiles,
		TotalPeers:    totalPeers,
		TotalSeeders:  totalSeeders,
		TotalLeechers: totalLeechers,
	}

	c.JSON(http.StatusOK, stats)
}

// 定期清理过期数据
func (ts *TrackerServer) startCleanupRoutine() {
	ticker := time.NewTicker(time.Duration(ts.config.CleanInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.cleanup()
		case <-ts.ctx.Done():
			return
		}
	}
}

// 清理过期数据
func (ts *TrackerServer) cleanup() {
	ctx := context.Background()

	// 获取所有文件key
	fileKeys, err := ts.redis.Keys(ctx, "file:*:peers").Result()
	if err != nil {
		ts.logger.Error("Failed to get file keys for cleanup", zap.Error(err))
		return
	}

	cleaned := 0
	for _, fileKey := range fileKeys {
		peerIDs, err := ts.redis.SMembers(ctx, fileKey).Result()
		if err != nil {
			continue
		}

		fileHash := strings.Split(fileKey, ":")[1]
		for _, peerID := range peerIDs {
			peerKey := ts.getPeerKey(fileHash, peerID)
			exists, err := ts.redis.Exists(ctx, peerKey).Result()
			if err != nil || exists == 0 {
				// Peer已过期，从集合中移除
				ts.redis.SRem(ctx, fileKey, peerID)
				cleaned++
			}
		}

		// 如果文件没有活跃peer，删除文件key
		count, _ := ts.redis.SCard(ctx, fileKey).Result()
		if count == 0 {
			ts.redis.Del(ctx, fileKey)
		}
	}

	if cleaned > 0 {
		ts.logger.Info("Cleanup completed", zap.Int("cleaned_peers", cleaned))
	}
}

// 启动服务器
func (ts *TrackerServer) Start() error {
	// 启动清理协程
	go ts.startCleanupRoutine()

	// 设置Gin路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// API路由
	api := r.Group("/api/v1")
	{
		api.POST("/announce", ts.handleAnnounce)
		api.GET("/scrape", ts.handleScrape)
		api.GET("/stats", ts.handleStats)
	}

	// 兼容性路由（直接在根路径）
	r.POST("/announce", ts.handleAnnounce)
	r.GET("/scrape", ts.handleScrape)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Prometheus指标
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 启动服务器
	addr := fmt.Sprintf(":%d", ts.config.ServerPort)
	ts.logger.Info("Starting tracker server", zap.String("address", addr))

	return r.Run(addr)
}

// 停止服务器
func (ts *TrackerServer) Stop() {
	ts.cancel()
	ts.redis.Close()
	ts.logger.Sync()
}

// 主函数
func main() {
	// 默认配置
	config := &Config{
		ServerPort:     8080,
		RedisAddr:      "localhost:6379",
		RedisPassword:  "",
		RedisDB:        0,
		CleanInterval:  300,  // 5分钟
		PeerTTL:        1800, // 30分钟
		MaxPeersReturn: 50,
	}

	// 创建tracker服务器
	server, err := NewTrackerServer(config)
	if err != nil {
		log.Fatalf("Failed to create tracker server: %v", err)
	}

	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
