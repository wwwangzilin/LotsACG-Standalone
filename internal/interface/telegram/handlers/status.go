package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/version"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// botStartedAt 记录进程启动时间, 用于显示运行时长。
var botStartedAt = time.Now()

// Status 处理 /status 指令: 显示机器人当前状态。
func Status(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("📊 <b>LotsACG 状态</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━\n")

	// 版本信息
	commit := version.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	sb.WriteString("📦 版本: <code>" + utils.EscapeHTML(version.Version) + "</code>\n")
	sb.WriteString("   commit: <code>" + utils.EscapeHTML(commit) + "</code>\n")
	sb.WriteString("   构建: <code>" + utils.EscapeHTML(version.BuildTime) + "</code>\n")
	sb.WriteString("⏱ 运行时长: <code>" + formatDuration(time.Since(botStartedAt)) + "</code>\n")

	// 作品统计
	total, err := serv.CountArtworks(ctx, shared.R18TypeAll)
	if err != nil {
		log.Warn("status: count artworks failed", "err", err)
	}
	r18, err := serv.CountArtworks(ctx, shared.R18TypeR18)
	if err != nil {
		log.Warn("status: count r18 artworks failed", "err", err)
	}
	cached, err := serv.CountCachedArtworks(ctx)
	if err != nil {
		log.Warn("status: count cached artworks failed", "err", err)
	}
	sb.WriteString("━━━━━━━━━━━━━━━━\n")
	sb.WriteString("📤 已发布作品: <code>" + fmt.Sprint(total) + "</code>\n")
	sb.WriteString("   (其中 R18: <code>" + fmt.Sprint(r18) + "</code>)\n")
	sb.WriteString("🗃 缓存作品: <code>" + fmt.Sprint(cached) + "</code>\n")

	// 发布队列
	sb.WriteString("━━━━━━━━━━━━━━━━\n")
	sb.WriteString("📨 发布队列: ")
	if q, err := loadPostQueue(ctx); err == nil && q != nil {
		switch q.Status {
		case postQueueStatusPosting:
			sb.WriteString("<code>进行中</code> (" + fmt.Sprint(len(q.Published)) + "/" + fmt.Sprint(len(q.SourceURLs)) + " 已发布)\n")
		case postQueueStatusCancelled:
			sb.WriteString("<code>已取消</code> (本次发布 " + fmt.Sprint(len(q.Published)) + " 条)\n")
		default:
			sb.WriteString("<code>空闲</code> (最近一次 " + fmt.Sprint(len(q.Published)) + "/" + fmt.Sprint(len(q.SourceURLs)) + ")\n")
		}
	} else {
		sb.WriteString("<code>无记录</code>\n")
	}

	// 图源
	sources := serv.Sources()
	names := make([]string, 0, len(sources))
	for st := range sources {
		names = append(names, string(st))
	}
	sort.Strings(names)
	sb.WriteString("🔌 图源: <code>" + utils.EscapeHTML(strings.Join(names, ", ")) + "</code>\n")

	utils.ReplyMessage(ctx, message, sb.String())
	return nil
}

// formatDuration 将时长格式化为 "Xh Ym Zs"。
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}
