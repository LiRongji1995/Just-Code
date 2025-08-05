package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobStatus 表示任务状态
type JobStatus string

const (
	JobStatusPending     JobStatus = "pending"     // 等待开始
	JobStatusMetadata    JobStatus = "metadata"    // 解析元数据
	JobStatusConnecting  JobStatus = "connecting"  // 连接Peer
	JobStatusDownloading JobStatus = "downloading" // 下载中
	JobStatusSeeding     JobStatus = "seeding"     // 做种中
	JobStatusCompleted   JobStatus = "completed"   // 已完成
	JobStatusFailed      JobStatus = "failed"      // 失败
	JobStatusPaused      JobStatus = "paused"      // 已暂停
)

// Job 表示一个下载或上传任务
// 设计思想：
// 1. 通过只读channel暴露状态，确保外部无法直接修改内部状态
// 2. 每个任务都有独立的生命周期管理
// 3. 支持暂停、恢复、取消等操作
type Job struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	// 只读channel，外部通过这些channel接收事件
	progressCh chan ProgressUpdate
	errorCh    chan *JobError
	doneCh     chan struct{}

	// 内部状态管理
	mu       sync.RWMutex
	status   JobStatus
	progress ProgressUpdate

	// 任务配置
	metaFile  string
	outputDir string
	engine    *Engine
}

// Progress 返回进度更新的只读channel
func (j *Job) Progress() <-chan ProgressUpdate {
	return j.progressCh
}

// Errors 返回错误事件的只读channel
func (j *Job) Errors() <-chan *JobError {
	return j.errorCh
}

// Done 返回完成信号的只读channel
// 当任务完成（成功或失败）时，这个channel会被关闭
func (j *Job) Done() <-chan struct{} {
	return j.doneCh
}

// ID 返回任务唯一标识符
func (j *Job) ID() string {
	return j.id
}

// Status 返回当前任务状态（线程安全）
func (j *Job) Status() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.status
}

// CurrentProgress 返回当前进度快照（线程安全）
func (j *Job) CurrentProgress() ProgressUpdate {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.progress
}

// MetaFile 返回元数据文件路径
func (j *Job) MetaFile() string {
	return j.metaFile
}

// OutputDir 返回输出目录
func (j *Job) OutputDir() string {
	return j.outputDir
}

// Pause 暂停任务
func (j *Job) Pause() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.status != JobStatusDownloading && j.status != JobStatusSeeding {
		return fmt.Errorf("cannot pause job in status: %s", j.status)
	}

	j.status = JobStatusPaused
	j.progress.Status = JobStatusPaused

	// 实际的暂停逻辑会在这里实现
	j.sendProgressUnsafe("任务已暂停")
	return nil
}

// Resume 恢复任务
func (j *Job) Resume() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.status != JobStatusPaused {
		return fmt.Errorf("cannot resume job in status: %s", j.status)
	}

	// 根据任务类型决定恢复到哪个状态
	if j.outputDir != "" {
		j.status = JobStatusDownloading
		j.progress.Status = JobStatusDownloading
	} else {
		j.status = JobStatusSeeding
		j.progress.Status = JobStatusSeeding
	}

	// 实际的恢复逻辑会在这里实现
	j.sendProgressUnsafe("任务已恢复")
	return nil
}

// Cancel 取消任务
func (j *Job) Cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.status == JobStatusCompleted || j.status == JobStatusFailed {
		return // 已经结束的任务无需取消
	}

	j.status = JobStatusFailed
	j.progress.Status = JobStatusFailed
	j.sendProgressUnsafe("任务已取消")

	j.cancel()
}

// 内部方法：发送进度更新（需要外部加锁）
func (j *Job) sendProgressUnsafe(message string) {
	j.progress.Message = message
	j.progress.Timestamp = time.Now()

	select {
	case j.progressCh <- j.progress:
	case <-j.ctx.Done():
		return
	default:
		// channel满了，跳过这次更新
	}
}

// 内部方法：发送进度更新（线程安全）
func (j *Job) sendProgress(message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.sendProgressUnsafe(message)
}

// 内部方法：更新进度数据（线程安全）
func (j *Job) updateProgress(updateFunc func(*ProgressUpdate)) {
	j.mu.Lock()
	defer j.mu.Unlock()

	updateFunc(&j.progress)
	j.progress.Timestamp = time.Now()

	select {
	case j.progressCh <- j.progress:
	case <-j.ctx.Done():
		return
	default:
		// channel满了，跳过这次更新
	}
}

// 内部方法：更新任务状态
func (j *Job) updateStatus(newStatus JobStatus, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.status = newStatus
	j.progress.Status = newStatus
	j.sendProgressUnsafe(message)
}

// 内部方法：发送错误
func (j *Job) sendError(errType, message string, fatal bool) {
	err := &JobError{
		JobID:     j.id,
		Type:      errType,
		Message:   message,
		Fatal:     fatal,
		Timestamp: time.Now(),
	}

	select {
	case j.errorCh <- err:
	case <-j.ctx.Done():
		return
	default:
		// channel满了，跳过这次错误
	}
}

// 内部方法：标记任务完成
func (j *Job) markDone() {
	close(j.doneCh)

	// 给其他channel一些时间来处理最后的消息
	time.AfterFunc(100*time.Millisecond, func() {
		close(j.progressCh)
		close(j.errorCh)
	})
}

// 内部方法：检查任务是否应该停止
func (j *Job) shouldStop() bool {
	select {
	case <-j.ctx.Done():
		return true
	default:
		return false
	}
}

// 内部方法：等待指定时间或取消信号
func (j *Job) waitOrCancel(duration time.Duration) bool {
	select {
	case <-time.After(duration):
		return false // 正常等待结束
	case <-j.ctx.Done():
		return true // 被取消
	}
}
