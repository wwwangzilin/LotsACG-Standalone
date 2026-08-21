package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// ArtworkNotifier 用于向用户推送新作品通知。
type ArtworkNotifier interface {
	// SendArtworkNotification 向用户发送新作品 (图片+说明)。
	SendArtworkNotification(ctx context.Context, userID int64, sourceURL string) error
	// SendTextToUser 向用户发送一条文本消息 (用于管理端提醒)。
	SendTextToUser(ctx context.Context, userID int64, text string) error
}

// inactiveArtistDays 画师超过该天数无新作品时向管理员发出提醒。
const inactiveArtistDays = 30

// StartFollowWatcher 定时检查画师关注与标签订阅, 发现新作品后推送给订阅用户。
func StartFollowWatcher(ctx context.Context, serv *service.Service, notifier ArtworkNotifier, interval time.Duration) {
	if serv == nil || notifier == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	doTask := func() {
		taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		checkArtistFollows(taskCtx, serv, notifier)
		checkTagSubscriptions(taskCtx, serv, notifier)
		checkInactiveArtists(taskCtx, serv, notifier)
	}
	doTask()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doTask()
		}
	}
}

// checkArtistFollows 检查所有被关注画师的新作品并通知订阅者。
func checkArtistFollows(ctx context.Context, serv *service.Service, notifier ArtworkNotifier) {
	artists, err := serv.AllFollowedArtists(ctx)
	if err != nil {
		log.Error("watcher: failed to list followed artists", "err", err)
		return
	}
	for i := range artists {
		data := &artists[i]
		urls, err := serv.FetchArtistArtworks(ctx, data.URL, 0)
		if err != nil {
			log.Warn("watcher: failed to fetch artist artworks", "url", data.URL, "err", err)
			continue
		}
		newURLs := diffStrings(data.SeenURLs, urls)
		if len(newURLs) == 0 {
			continue
		}
		data.SeenURLs = unionStrings(data.SeenURLs, urls)
		data.LastActive = time.Now().Unix()
		if err := serv.SaveArtistFollowData(ctx, data); err != nil {
			log.Warn("watcher: failed to save artist follow data", "url", data.URL, "err", err)
		}
		for _, userID := range data.Users {
			for _, url := range newURLs {
				if err := notifier.SendArtworkNotification(ctx, userID, url); err != nil {
					log.Warn("watcher: failed to notify user", "user", userID, "url", url, "err", err)
				}
			}
		}
		log.Info("watcher: artist new artworks notified", "url", data.URL, "new", len(newURLs), "users", len(data.Users))
	}
}

// checkTagSubscriptions 检查所有被订阅标签的新作品并通知订阅者。
func checkTagSubscriptions(ctx context.Context, serv *service.Service, notifier ArtworkNotifier) {
	subs, err := serv.AllSubscribedTags(ctx)
	if err != nil {
		log.Error("watcher: failed to list tag subscriptions", "err", err)
		return
	}
	for i := range subs {
		data := &subs[i]
		arts, err := serv.SearchNewArtworksByTagsOrdered(ctx, []string{data.Tag}, 30, "date_d")
		if err != nil {
			log.Warn("watcher: failed to search tag", "tag", data.Tag, "err", err)
		}
		newURLs := make([]string, 0)
		for _, a := range arts {
			if a == nil || a.SourceURL == "" {
				continue
			}
			if !containsString(data.SeenURLs, a.SourceURL) {
				newURLs = append(newURLs, a.SourceURL)
				data.SeenURLs = append(data.SeenURLs, a.SourceURL)
			}
		}
		if len(newURLs) == 0 {
			continue
		}
		if err := serv.SaveTagSubscribeData(ctx, data); err != nil {
			log.Warn("watcher: failed to save tag subscribe data", "tag", data.Tag, "err", err)
		}
		for _, userID := range data.Users {
			for _, url := range newURLs {
				if err := notifier.SendArtworkNotification(ctx, userID, url); err != nil {
					log.Warn("watcher: failed to notify user", "user", userID, "url", url, "err", err)
				}
			}
		}
		log.Info("watcher: tag new artworks notified", "tag", data.Tag, "new", len(newURLs), "users", len(data.Users))
	}
}

// checkInactiveArtists 检查长期无新作品的关注画师 (默认 30 天), 向管理员发送提醒。
// 避免刷屏: 同一画师 30 天内最多提醒一次 (记录在 data.LastAlert)。
func checkInactiveArtists(ctx context.Context, serv *service.Service, notifier ArtworkNotifier) {
	artists, err := serv.AllFollowedArtists(ctx)
	if err != nil {
		log.Error("watcher: failed to list followed artists for inactive check", "err", err)
		return
	}
	now := time.Now()
	inactive := make([]*service.ArtistFollowData, 0)
	for i := range artists {
		data := &artists[i]
		if data.FirstFollow <= 0 {
			continue // 无首次关注时间 (历史数据), 不参与判断
		}
		last := data.LastActive
		if last <= 0 {
			last = data.FirstFollow
		}
		if now.Unix()-last < inactiveArtistDays*24*3600 {
			continue // 最近还有活动
		}
		if data.LastAlert > 0 && now.Unix()-data.LastAlert < inactiveArtistDays*24*3600 {
			continue // 30 天内已提醒过
		}
		inactive = append(inactive, data)
	}
	if len(inactive) == 0 {
		return
	}
	admins, err := serv.GetAdminUserIDs(ctx)
	if err != nil || len(admins) == 0 {
		log.Warn("watcher: no admin to notify inactive artists", "err", err)
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "⚠️ 关注画师长期无更新 (超过 %d 天):\n", inactiveArtistDays)
	for _, d := range inactive {
		days := (now.Unix() - d.LastActive) / 86400
		if d.LastActive <= 0 {
			days = (now.Unix() - d.FirstFollow) / 86400
		}
		fmt.Fprintf(&b, "\n• %s (约 %d 天无更新)", d.URL, days)
	}
	text := b.String()
	for _, adminID := range admins {
		if err := notifier.SendTextToUser(ctx, adminID, text); err != nil {
			log.Warn("watcher: failed to notify admin inactive artists", "admin", adminID, "err", err)
		}
	}
	for _, d := range inactive {
		d.LastAlert = now.Unix()
		if err := serv.SaveArtistFollowData(ctx, d); err != nil {
			log.Warn("watcher: failed to save last_alert", "url", d.URL, "err", err)
		}
	}
	log.Info("watcher: inactive artists notified", "count", len(inactive))
}

// diffStrings 返回 base 中不存在于 list 的元素 (保持 list 顺序)。
func diffStrings(base, list []string) []string {
	set := make(map[string]struct{}, len(base))
	for _, s := range base {
		set[s] = struct{}{}
	}
	out := make([]string, 0)
	for _, s := range list {
		if _, ok := set[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

// unionStrings 返回 base 与 list 的并集 (保持顺序, 去重)。
func unionStrings(base, list []string) []string {
	set := make(map[string]struct{}, len(base))
	out := make([]string, 0, len(base)+len(list))
	for _, s := range base {
		if _, ok := set[s]; ok {
			continue
		}
		set[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range list {
		if _, ok := set[s]; ok {
			continue
		}
		set[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
