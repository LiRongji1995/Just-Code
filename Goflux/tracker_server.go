package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
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
	config     *Config
	redis      *redis.Client
	logger     *zap.Logger
	metrics    *TrackerMetrics
	ctx        context.Context
	cancel     context.CancelFunc
	httpServer *http.Server
	wg         sync.WaitGroup
}

// Prometheus指标
type TrackerMetrics struct {
	announceRequests prometheus.Counter
	scrapeRequests   prometheus.Counter
	activePeers      prometheus.Gauge
	activeFiles      prometheus.Gauge
	errorCounter     prometheus.Counter

	// 新增指标
	blacklistedRequests prometheus.Counter
	responseTime        prometheus.Histogram
	peersByType         *prometheus.GaugeVec // seeders vs leechers
	topFiles            prometheus.Gauge     // 热门文件数量
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
		blacklistedRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tracker_blacklisted_requests_total",
			Help: "Total number of blocked requests due to blacklist",
		}),
		responseTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tracker_response_time_seconds",
			Help:    "Response time of tracker requests",
			Buckets: prometheus.DefBuckets, // 默认桶
		}),
		peersByType: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tracker_peers_by_type",
			Help: "Number of peers by type (seeder/leecher)",
		}, []string{"type"}),
		topFiles: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tracker_top_files",
			Help: "Number of files in top rankings",
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
		m.blacklistedRequests,
		m.responseTime,
		m.peersByType,
		m.topFiles,
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

func (ts *TrackerServer) getIPBlacklistKey() string {
	return "tracker:ip_blacklist"
}

func (ts *TrackerServer) getFileStatsZSetKey() string {
	return "tracker:file_stats"
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

	// IP黑名单检查
	if ts.isIPBlacklisted(req.IP) {
		ts.metrics.blacklistedRequests.Inc()
		ts.logger.Warn("Blocked IP attempt", zap.String("ip", req.IP))
		c.JSON(http.StatusForbidden, gin.H{"error": "IP blocked"})
		return
	}

	// 限制每个文件的最大peer数
	if ts.exceedsMaxPeers(req.FileHash) {
		ts.logger.Warn("File has too many peers", zap.String("file_hash", req.FileHash))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "File has reached maximum peer limit"})
		return
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

	// 获取peer列表（使用nearest peer策略）
	peers, err := ts.getPeersOptimized(ctx, req.FileHash, req.PeerID, req.IP, req.NumWant)
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

	// 使用HSET优化：把每个file的所有peer放在一个hash中，减少key数量
	fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)
	if err := ts.redis.HSet(ctx, fileHashKey, peer.PeerID, peerData).Err(); err != nil {
		return err
	}

	// 设置hash的TTL
	ts.redis.Expire(ctx, fileHashKey, time.Duration(ts.config.PeerTTL)*time.Second)

	// 使用timestamp进行排序，便于查找最新的peer
	timestampZSet := fmt.Sprintf("file:%s:peer_timestamps", fileHash)
	ts.redis.ZAdd(ctx, timestampZSet, redis.Z{
		Score:  float64(peer.LastSeen.Unix()),
		Member: peer.PeerID,
	})
	ts.redis.Expire(ctx, timestampZSet, time.Duration(ts.config.PeerTTL)*time.Second)

	// 更新文件统计的ZSet，用于快速获取热门文件
	fileStatsKey := ts.getFileStatsZSetKey()
	ts.redis.ZIncrBy(ctx, fileStatsKey, 1, fileHash)

	// 更新统计信息
	ts.updateFileStats(ctx, fileHash, peer)

	return nil
}

// 移除peer
func (ts *TrackerServer) removePeer(ctx context.Context, fileHash, peerID string) error {
	// 从hash中删除peer信息
	fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)
	ts.redis.HDel(ctx, fileHashKey, peerID)

	// 从timestamp ZSet中移除
	timestampZSet := fmt.Sprintf("file:%s:peer_timestamps", fileHash)
	ts.redis.ZRem(ctx, timestampZSet, peerID)

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

// 优化的获取peer列表方法（使用HGETALL和ZSet排序）
func (ts *TrackerServer) getPeersOptimized(ctx context.Context, fileHash, excludePeerID, requestIP string, numWant int) ([]PeerInfo, error) {
	// 使用HGETALL一次性获取所有peer数据
	fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)
	allPeerData, err := ts.redis.HGetAll(ctx, fileHashKey).Result()
	if err != nil {
		return nil, err
	}

	var allPeers []PeerInfo
	for peerID, peerDataStr := range allPeerData {
		if peerID == excludePeerID {
			continue
		}

		var peer PeerInfo
		if err := json.Unmarshal([]byte(peerDataStr), &peer); err != nil {
			continue
		}
		allPeers = append(allPeers, peer)
	}

	// 实现nearest peer策略：优先返回同网段的peer
	peers := ts.selectNearestPeers(allPeers, requestIP, numWant)

	return peers, nil
}

// IP黑名单检查 - 增强版
func (ts *TrackerServer) isIPBlacklisted(ip string) bool {
	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	// 检查完整IP
	exists, err := ts.redis.SIsMember(ctx, blacklistKey, ip).Result()
	if err != nil || exists {
		return exists
	}

	// 检查各级网段匹配
	ipParts := strings.Split(ip, ".")
	if len(ipParts) != 4 {
		return false
	}

	// 检查 /24 网段 (192.168.1.*)
	subnet24 := fmt.Sprintf("%s.%s.%s.*", ipParts[0], ipParts[1], ipParts[2])
	if exists, _ := ts.redis.SIsMember(ctx, blacklistKey, subnet24).Result(); exists {
		return true
	}

	// 检查 /16 网段 (192.168.*.*)
	subnet16 := fmt.Sprintf("%s.%s.*.*", ipParts[0], ipParts[1])
	if exists, _ := ts.redis.SIsMember(ctx, blacklistKey, subnet16).Result(); exists {
		return true
	}

	// 检查 /8 网段 (192.*.*.*)
	subnet8 := fmt.Sprintf("%s.*.*.*", ipParts[0])
	if exists, _ := ts.redis.SIsMember(ctx, blacklistKey, subnet8).Result(); exists {
		return true
	}

	return false
}

// 检查文件是否超过最大peer数限制
func (ts *TrackerServer) exceedsMaxPeers(fileHash string) bool {
	ctx := context.Background()
	fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)

	count, err := ts.redis.HLen(ctx, fileHashKey).Result()
	if err != nil {
		return false
	}

	maxPeersPerFile := 1000 // 可配置
	return count >= int64(maxPeersPerFile)
}

// 选择最近的peer（网络距离）- 增强版
func (ts *TrackerServer) selectNearestPeers(allPeers []PeerInfo, requestIP string, numWant int) []PeerInfo {
	if len(allPeers) <= numWant {
		return allPeers
	}

	// 按网络距离和peer质量排序
	type peerWithScore struct {
		peer     PeerInfo
		distance int
		score    float64 // 综合评分
	}

	var peersWithScore []peerWithScore
	requestIPParts := strings.Split(requestIP, ".")

	for _, peer := range allPeers {
		distance := ts.calculateIPDistance(requestIPParts, strings.Split(peer.IP, "."))

		// 计算综合评分：考虑网络距离、上传量、活跃度
		score := ts.calculatePeerScore(peer, distance)

		peersWithScore = append(peersWithScore, peerWithScore{
			peer:     peer,
			distance: distance,
			score:    score,
		})
	}

	// 按综合评分排序（评分越高越好）
	sort.Slice(peersWithScore, func(i, j int) bool {
		return peersWithScore[i].score > peersWithScore[j].score
	})

	// 策略：70%选择高分peer，30%随机选择（保持网络多样性）
	var result []PeerInfo
	highScoreCount := numWant * 7 / 10
	randomCount := numWant - highScoreCount

	// 添加高分peer
	for i := 0; i < highScoreCount && i < len(peersWithScore); i++ {
		result = append(result, peersWithScore[i].peer)
	}

	// 随机添加一些peer保持网络健康
	if randomCount > 0 && len(peersWithScore) > highScoreCount {
		remaining := peersWithScore[highScoreCount:]
		rand.Shuffle(len(remaining), func(i, j int) {
			remaining[i], remaining[j] = remaining[j], remaining[i]
		})

		for i := 0; i < randomCount && i < len(remaining); i++ {
			result = append(result, remaining[i].peer)
		}
	}

	return result
}

// 计算peer综合评分
func (ts *TrackerServer) calculatePeerScore(peer PeerInfo, distance int) float64 {
	// 网络距离评分（距离越近分数越高）
	networkScore := float64(4-distance) * 25.0 // 0-100分

	// 上传贡献评分（鼓励上传）
	uploadScore := 0.0
	if peer.Uploaded > 0 {
		uploadScore = math.Min(float64(peer.Uploaded)/1024/1024/100, 20.0) // 最多20分，每100MB得1分
	}

	// 活跃度评分（最近活跃时间）
	activeScore := 0.0
	timeSinceLastSeen := time.Since(peer.LastSeen).Minutes()
	if timeSinceLastSeen < 5 {
		activeScore = 15.0 // 5分钟内活跃，满分
	} else if timeSinceLastSeen < 30 {
		activeScore = 10.0 // 30分钟内活跃
	} else {
		activeScore = 5.0 // 其他情况
	}

	// Seeder优先（完整文件的peer）
	seederBonus := 0.0
	if peer.Left == 0 {
		seederBonus = 10.0
	}

	return networkScore + uploadScore + activeScore + seederBonus
}

// 计算IP距离（简单的网段匹配）
func (ts *TrackerServer) calculateIPDistance(ip1Parts, ip2Parts []string) int {
	if len(ip1Parts) != 4 || len(ip2Parts) != 4 {
		return 4 // 最大距离
	}

	distance := 0
	for i := 0; i < 4; i++ {
		if ip1Parts[i] != ip2Parts[i] {
			distance = 4 - i
			break
		}
	}
	return distance
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
	fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)

	// 使用HGETALL一次性获取所有peer
	allPeerData, err := ts.redis.HGetAll(ctx, fileHashKey).Result()
	if err != nil {
		return nil, err
	}

	var seeders, leechers int
	for _, peerDataStr := range allPeerData {
		var peer PeerInfo
		if err := json.Unmarshal([]byte(peerDataStr), &peer); err != nil {
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

// 优化后的统计信息处理 - 使用ZSCAN避免KEYS命令
func (ts *TrackerServer) handleStats(c *gin.Context) {
	ctx := context.Background()

	// 使用ZSet来统计活跃文件
	fileStatsKey := ts.getFileStatsZSetKey()
	fileHashes, err := ts.redis.ZRevRange(ctx, fileStatsKey, 0, -1).Result()
	if err != nil {
		ts.metrics.errorCounter.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	totalFiles := len(fileHashes)
	totalPeers := 0
	totalSeeders := 0
	totalLeechers := 0

	// 遍历活跃文件计算统计信息
	for _, fileHash := range fileHashes {
		fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)
		allPeerData, err := ts.redis.HGetAll(ctx, fileHashKey).Result()
		if err != nil {
			continue
		}

		totalPeers += len(allPeerData)

		// 计算seeder和leecher
		for _, peerDataStr := range allPeerData {
			var peer PeerInfo
			if err := json.Unmarshal([]byte(peerDataStr), &peer); err != nil {
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
	ts.metrics.peersByType.WithLabelValues("seeder").Set(float64(totalSeeders))
	ts.metrics.peersByType.WithLabelValues("leecher").Set(float64(totalLeechers))

	stats := TrackerStats{
		TotalFiles:    totalFiles,
		TotalPeers:    totalPeers,
		TotalSeeders:  totalSeeders,
		TotalLeechers: totalLeechers,
	}

	c.JSON(http.StatusOK, stats)
}

// 优化后的清理程序 - 使用时间戳ZSet进行清理
func (ts *TrackerServer) startCleanupRoutine() {
	ts.wg.Add(1)
	defer ts.wg.Done()

	ticker := time.NewTicker(time.Duration(ts.config.CleanInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.cleanupOptimized()
		case <-ts.ctx.Done():
			ts.logger.Info("Cleanup routine stopping...")
			return
		}
	}
}

// 优化的清理方法
func (ts *TrackerServer) cleanupOptimized() {
	ctx := context.Background()

	// 获取所有活跃文件的ZSet
	fileStatsKey := ts.getFileStatsZSetKey()
	fileHashes, err := ts.redis.ZRevRange(ctx, fileStatsKey, 0, -1).Result()
	if err != nil {
		ts.logger.Error("Failed to get file hashes for cleanup", zap.Error(err))
		return
	}

	cleaned := 0
	expiredFiles := 0

	// 计算过期时间戳
	expiredTimestamp := time.Now().Add(-time.Duration(ts.config.PeerTTL) * time.Second).Unix()

	for _, fileHash := range fileHashes {
		// 使用ZREMRANGEBYSCORE清理过期的peer
		timestampZSet := fmt.Sprintf("file:%s:peer_timestamps", fileHash)
		removedCount, err := ts.redis.ZRemRangeByScore(ctx, timestampZSet, "0", fmt.Sprintf("%d", expiredTimestamp)).Result()
		if err != nil {
			continue
		}

		if removedCount > 0 {
			cleaned += int(removedCount)

			// 获取过期的peer ID并从hash中删除
			fileHashKey := fmt.Sprintf("file:%s:peer_data", fileHash)

			// 如果所有peer都过期了，删除整个文件的数据
			remainingCount, _ := ts.redis.ZCard(ctx, timestampZSet).Result()
			if remainingCount == 0 {
				ts.redis.Del(ctx, fileHashKey, timestampZSet)
				ts.redis.ZRem(ctx, fileStatsKey, fileHash)
				expiredFiles++
			}
		}
	}

	if cleaned > 0 || expiredFiles > 0 {
		ts.logger.Info("Cleanup completed",
			zap.Int("cleaned_peers", cleaned),
			zap.Int("expired_files", expiredFiles))
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

		// 管理接口
		admin := api.Group("/admin")
		{
			admin.POST("/blacklist/ip", ts.handleAddIPToBlacklist)
			admin.POST("/blacklist/batch", ts.handleBatchAddToBlacklist)
			admin.DELETE("/blacklist/ip", ts.handleRemoveIPFromBlacklist)
			admin.GET("/blacklist/ip", ts.handleGetBlacklist)
			admin.DELETE("/blacklist/clear", ts.handleClearBlacklist)
		}
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

	// 创建HTTP服务器
	ts.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", ts.config.ServerPort),
		Handler: r,
	}

	ts.logger.Info("Starting tracker server", zap.String("address", ts.httpServer.Addr))

	// 启动服务器
	return ts.httpServer.ListenAndServe()
}

// 优雅关闭服务器
func (ts *TrackerServer) Shutdown() error {
	ts.logger.Info("Shutting down tracker server...")

	// 取消context，停止所有协程
	ts.cancel()

	// 等待清理协程结束
	ts.wg.Wait()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ts.httpServer.Shutdown(ctx); err != nil {
		ts.logger.Error("Failed to shutdown HTTP server gracefully", zap.Error(err))
		return err
	}

	// 关闭Redis连接
	if err := ts.redis.Close(); err != nil {
		ts.logger.Error("Failed to close Redis connection", zap.Error(err))
	}

	// 同步日志
	ts.logger.Sync()

	ts.logger.Info("Tracker server shutdown completed")
	return nil
}

// IP黑名单管理接口
func (ts *TrackerServer) handleAddIPToBlacklist(c *gin.Context) {
	var req struct {
		IP string `json:"ip" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	if err := ts.redis.SAdd(ctx, blacklistKey, req.IP).Err(); err != nil {
		ts.logger.Error("Failed to add IP to blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add IP"})
		return
	}

	ts.logger.Info("IP added to blacklist", zap.String("ip", req.IP))
	c.JSON(http.StatusOK, gin.H{"message": "IP added to blacklist"})
}

func (ts *TrackerServer) handleRemoveIPFromBlacklist(c *gin.Context) {
	var req struct {
		IP string `json:"ip" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	if err := ts.redis.SRem(ctx, blacklistKey, req.IP).Err(); err != nil {
		ts.logger.Error("Failed to remove IP from blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove IP"})
		return
	}

	ts.logger.Info("IP removed from blacklist", zap.String("ip", req.IP))
	c.JSON(http.StatusOK, gin.H{"message": "IP removed from blacklist"})
}

func (ts *TrackerServer) handleGetBlacklist(c *gin.Context) {
	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	ips, err := ts.redis.SMembers(ctx, blacklistKey).Result()
	if err != nil {
		ts.logger.Error("Failed to get blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blacklist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklisted_ips": ips, "count": len(ips)})
}

// 批量添加黑名单
func (ts *TrackerServer) handleBatchAddToBlacklist(c *gin.Context) {
	var req struct {
		IPs []string `json:"ips" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	// 转换为interface{}切片
	ips := make([]interface{}, len(req.IPs))
	for i, ip := range req.IPs {
		ips[i] = ip
	}

	added, err := ts.redis.SAdd(ctx, blacklistKey, ips...).Result()
	if err != nil {
		ts.logger.Error("Failed to batch add IPs to blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add IPs"})
		return
	}

	ts.logger.Info("Batch added IPs to blacklist",
		zap.Int("requested", len(req.IPs)),
		zap.Int64("actually_added", added))

	c.JSON(http.StatusOK, gin.H{
		"message":   "IPs added to blacklist",
		"requested": len(req.IPs),
		"added":     added,
	})
}

// 清空黑名单
func (ts *TrackerServer) handleClearBlacklist(c *gin.Context) {
	ctx := context.Background()
	blacklistKey := ts.getIPBlacklistKey()

	count, err := ts.redis.Del(ctx, blacklistKey).Result()
	if err != nil {
		ts.logger.Error("Failed to clear blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear blacklist"})
		return
	}

	ts.logger.Info("Blacklist cleared", zap.Int64("deleted_keys", count))
	c.JSON(http.StatusOK, gin.H{"message": "Blacklist cleared"})
}

// 主函数 - 实现优雅关闭
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

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器的goroutine
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待关闭信号
	<-sigChan

	// 执行优雅关闭
	if err := server.Shutdown(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
