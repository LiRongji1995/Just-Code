package cmd

import (
	"goflux/engine"
	"path/filepath"
)

// GetStatusIcon 根据 JobStatus 返回一个 Unicode 图标
func getStatusIcon(status engine.JobStatus) string {
	switch status {
	case engine.JobStatusMetadata:
		return "📄"
	case engine.JobStatusConnecting:
		return "🔌"
	case engine.JobStatusDownloading:
		return "⬇️"
	case engine.JobStatusSeeding:
		return "📤"
	case engine.JobStatusCompleted:
		return "✅"
	case engine.JobStatusFailed:
		return "❌"
	case engine.JobStatusPaused:
		return "⏸️"
	default:
		return "❓"
	}
}

// getOutputPath 返回文件的保存完整路径
func getOutputPath(fileName string) string {
	return filepath.Join("downloads", fileName)
}
