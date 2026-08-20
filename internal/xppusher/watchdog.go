package xppusher

import (
	"context"
	"sync"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// ProcessManager 看门狗所需的子进程管理接口 (xppusher.Manager 与 kmua.Manager 均实现)。
type ProcessManager interface {
	Start(ctx context.Context, progress func(string)) (int, error)
	RunningPID(ctx context.Context) int
}

// Watchdog 子进程看门狗: 定期检查进程存活, 异常退出时自动重启。
// 同时统计连续重启次数, 超过上限后停止并告警, 避免死循环。
type Watchdog struct {
	manager     ProcessManager
	interval    time.Duration
	maxRestarts int

	mu         sync.Mutex
	restarts   int
	stopped    bool
	startedPID int
}

// NewWatchdog 创建看门狗。interval<=0 用默认 30s; maxRestarts<=0 用默认 5。
func NewWatchdog(m ProcessManager, interval time.Duration, maxRestarts int) *Watchdog {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if maxRestarts <= 0 {
		maxRestarts = 5
	}
	return &Watchdog{
		manager:     m,
		interval:    interval,
		maxRestarts: maxRestarts,
	}
}

// Run 启动看门狗循环, 直到 ctx 取消。
// 每次检查: 若之前已启动过且进程退出, 尝试自动重启 (受次数上限约束)。
func (w *Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check()
		}
	}
}

func (w *Watchdog) check() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	pid := w.manager.RunningPID(context.Background())
	alive := pid > 0
	w.mu.Unlock()

	// 进程还活着: 一切正常
	if alive {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	// 从未启动过 (startedPID==0) 或看门狗已停: 不自动拉起 (由 AutoStart 负责首启)
	if w.startedPID == 0 || w.stopped {
		return
	}
	// 连续重启达到上限: 停止并告警
	if w.restarts >= w.maxRestarts {
		w.stopped = true
		log.Errorf("watchdog: 连续重启 %d 次失败, 停止自动重启", w.restarts)
		return
	}
	w.restarts++
	log.Warnf("watchdog: 进程已退出 (pid=%d), 自动重启 (第 %d 次)", w.startedPID, w.restarts)
	newPid, err := w.manager.Start(context.Background(), nil)
	if err != nil {
		log.Errorf("watchdog: 重启失败: %v", err)
		return
	}
	w.startedPID = newPid
	log.Infof("watchdog: 已重启, pid=%d", newPid)
}

// MarkStarted 记录进程已由外部 (AutoStart / 手动) 启动, 看门狗开始监控该进程。
func (w *Watchdog) MarkStarted(pid int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.startedPID = pid
	w.restarts = 0
	w.stopped = false
}

// Stop 停止看门狗 (不再自动重启)。
func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
}

// RestartCount 返回累计重启次数。
func (w *Watchdog) RestartCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.restarts
}

// WatchdogEnabled 判断配置是否启用看门狗。
func WatchdogEnabled(cfg runtimecfg.XPPusherConfig) bool {
	if cfg.WatchdogEnable != nil {
		return *cfg.WatchdogEnable
	}
	return true // 默认启用
}
