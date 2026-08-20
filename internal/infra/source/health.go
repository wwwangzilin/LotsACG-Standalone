package source

import (
	"sync"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

// HealthStats 单源健康度统计。
type HealthStats struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// SuccessRate 返回成功率 (0~100), 无样本时返回 -1 表示未知。
func (h HealthStats) SuccessRate() float64 {
	total := h.Success + h.Failed
	if total == 0 {
		return -1
	}
	return float64(h.Success) / float64(total) * 100
}

// healthTracker 全局源健康度追踪器。
type healthTracker struct {
	mu     sync.RWMutex
	stats  map[shared.SourceType]*HealthStats
}

var tracker = &healthTracker{stats: make(map[shared.SourceType]*HealthStats)}

// RecordSuccess 记录一次源抓取成功。
func RecordSuccess(t shared.SourceType) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	s := tracker.stats[t]
	if s == nil {
		s = &HealthStats{}
		tracker.stats[t] = s
	}
	s.Success++
}

// RecordFailure 记录一次源抓取失败。
func RecordFailure(t shared.SourceType) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	s := tracker.stats[t]
	if s == nil {
		s = &HealthStats{}
		tracker.stats[t] = s
	}
	s.Failed++
}

// HealthSnapshot 返回所有源的健康度快照 (供 /status 展示)。
func HealthSnapshot() map[shared.SourceType]HealthStats {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	out := make(map[shared.SourceType]HealthStats, len(tracker.stats))
	for k, v := range tracker.stats {
		out[k] = *v
	}
	return out
}
