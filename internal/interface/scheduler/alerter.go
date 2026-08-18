package scheduler

import (
	"sync"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// Alerter 抓取错误告警器：统计各源连续失败，超过阈值时触发通知回调。
type Alerter struct {
	mu         sync.Mutex
	failures   map[string]int  // source -> 连续失败次数
	threshold  int             // 触发告警的连续失败阈值
	notify     func(source string, failures int) // 通知回调（发给管理员）
	lastNotify map[string]time.Time // source -> 上次告警时间（防止刷屏）
}

// NewAlerter 创建告警器。threshold <= 0 时不启用；cooldown 为同一源两次告警的最小间隔。
func NewAlerter(threshold int, cooldown time.Duration, notify func(source string, failures int)) *Alerter {
	if threshold <= 0 {
		return nil
	}
	return &Alerter{
		failures:   make(map[string]int),
		threshold:  threshold,
		notify:     notify,
		lastNotify: make(map[string]time.Time),
	}
}

// RecordFailure 记录一次抓取失败；达到阈值时触发告警。
func (a *Alerter) RecordFailure(source string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failures[source]++
	fails := a.failures[source]
	if fails >= a.threshold && a.notify != nil {
		last, ok := a.lastNotify[source]
		// 冷却期内不重复告警
		if !ok || time.Since(last) > 30*time.Minute {
			log.Warnf("alerter: source %s failed %d times consecutively", source, fails)
			go func() {
				defer func() { recover() }()
				a.notify(source, fails)
			}()
			a.lastNotify[source] = time.Now()
		}
	}
}

// RecordSuccess 记录一次成功抓取，重置该源的失败计数。
func (a *Alerter) RecordSuccess(source string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failures[source] = 0
}

// Snapshot 返回当前各源失败计数（供 /status 展示）。
func (a *Alerter) Snapshot() map[string]int {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[string]int, len(a.failures))
	for k, v := range a.failures {
		out[k] = v
	}
	return out
}
