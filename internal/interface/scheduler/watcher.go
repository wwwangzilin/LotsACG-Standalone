package scheduler

import (
	"context"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// ArtworkNotifier 用于向用户推送新作品通知。
type ArtworkNotifier interface {
	// SendArtworkNotification 向用户发送新作品 (图片+说明)。
	SendArtworkNotification(ctx context.Context, userID int64, sourceURL string) error
}

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
