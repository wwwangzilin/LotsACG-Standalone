package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
)

var (
	startTime = time.Now()

	// 指标：入库作品数、各源抓取数、抓取失败数、发布队列深度
	artworksTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lotsacg_artworks_total",
		Help: "Total artworks stored in the database.",
	})
	fetchTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lotsacg_fetch_total",
		Help: "Total artworks fetched per source.",
	}, []string{"source"})
	fetchErrorTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lotsacg_fetch_error_total",
		Help: "Total fetch errors per source.",
	}, []string{"source"})
	queueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lotsacg_queue_depth",
		Help: "Current depth of the artwork posting queue.",
	})
)

func init() {
	prometheus.MustRegister(artworksTotal, fetchTotal, fetchErrorTotal, queueDepth)
}

// MetricsCollector 供服务层上报指标（service 层通过它更新计数器）。
type MetricsCollector struct {
	Artworks func() float64 // 返回当前入库作品数，nil 时跳过
	Queue    func() float64 // 返回当前队列深度，nil 时跳过
}

var collector MetricsCollector

// RegisterMetricsCollector 由 server 装配时调用，注入指标读取函数。
func RegisterMetricsCollector(c MetricsCollector) {
	collector = c
}

// UpdateMetrics 周期性同步外部指标到 prometheus gauge。
func UpdateMetrics() {
	if collector.Queue != nil {
		queueDepth.Set(collector.Queue())
	}
	if collector.Artworks != nil {
		artworksTotal.Add(collector.Artworks())
	}
}

// HandleHealthz 存活探针：进程活着即返回 200。
func HandleHealthz(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(common.NewSuccess(map[string]any{
		"status": "ok",
		"uptime": time.Since(startTime).String(),
	}))
}

// HandleReadyz 就绪探针：检查依赖（bot 是否启用、数据库可用性），失败返回 503。
func HandleReadyz(ctx fiber.Ctx) error {
	_, botOK := common.GetState[common.TelegramBot](ctx, common.StateKeyTelegramBot)
	checks := map[string]bool{
		"bot_enabled": botOK,
	}
	if !botOK {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(common.NewError(
			fiber.StatusServiceUnavailable,
			"not ready: telegram bot is not enabled",
		).Response())
	}
	return ctx.Status(fiber.StatusOK).JSON(common.NewSuccess(checks))
}

// HandleMetrics Prometheus 指标端点（通过 adaptor 桥接原生 promhttp.Handler）。
func HandleMetrics(ctx fiber.Ctx) error {
	UpdateMetrics()
	return adaptor.HTTPHandler(promhttp.Handler())(ctx)
}

// HandleStatusEx 增强版状态：运行时信息 + 依赖状态。
func HandleStatusEx(ctx fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := map[string]any{
		"uptime":      time.Since(startTime).String(),
		"go_version":  runtime.Version(),
		"goroutines":  runtime.NumGoroutine(),
		"memory_mb":   m.Alloc / 1024 / 1024,
		"num_cpu":     runtime.NumCPU(),
		"started_at":  startTime.Format(time.RFC3339),
		"bot_running": false,
	}
	if collector.Queue != nil {
		status["queue_depth"] = collector.Queue()
	}
	if collector.Artworks != nil {
		status["artworks"] = collector.Artworks()
	}

	if bot, ok := common.GetState[common.TelegramBot](ctx, common.StateKeyTelegramBot); ok {
		s := bot.Status(context.Background())
		status["bot_running"] = s.Running
		status["bot_username"] = s.BotUsername
	}

	return ctx.JSON(common.NewSuccess(status))
}
