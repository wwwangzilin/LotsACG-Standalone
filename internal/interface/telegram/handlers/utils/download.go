package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/imgcandidates"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/pixiv"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/twitter"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
)

// DownloadPixivImageWithFallback downloads a Pixiv image trying the configured
// proxy hosts in priority order (manyacg -> pixiv.cat -> i.muxmus.com -> official).
// Returns the downloaded file and the URL that succeeded.
func DownloadPixivImageWithFallback(ctx context.Context, imageURL string, attemptsPerSource int) (*osutil.File, string, error) {
	proxyHosts := runtimecfg.Get().Source.Pixiv.ImgProxyHosts()
	return DownloadImageWithFallback(ctx, imageURL, proxyHosts, attemptsPerSource, pixiv.BuildPixivImageCandidates)
}

// DownloadTwitterImageWithFallback downloads a Twitter image trying the
// configured twimg reverse-proxy hosts in priority order, official
// pbs.twimg.com last. Non-twimg URLs are downloaded directly.
func DownloadTwitterImageWithFallback(ctx context.Context, imageURL string, attemptsPerSource int) (*osutil.File, string, error) {
	proxyHosts := runtimecfg.Get().Source.Twitter.ImgProxyHosts()
	return DownloadImageWithFallback(ctx, imageURL, proxyHosts, attemptsPerSource, twitter.BuildTwitterImageCandidates)
}

// DownloadImageWithFallback tries downloading an image over the candidate URLs
// built by build (proxies first, official domain last), with retries per source.
func DownloadImageWithFallback(ctx context.Context, imageURL string, proxyHosts []string, attemptsPerSource int, build func(string, []string) []string) (*osutil.File, string, error) {
	if attemptsPerSource <= 0 {
		attemptsPerSource = 3
	}
	candidates := build(imageURL, proxyHosts)

	var lastErr error
	for ci, candidate := range candidates {
		for retry := 0; retry < attemptsPerSource; retry++ {
			if retry > 0 {
				time.Sleep(time.Duration(retry) * time.Second)
			}
			f, err := httpclient.DownloadWithCache(ctx, candidate, nil)
			if err == nil {
				return f, candidate, nil
			}
			lastErr = err
			log.Warnf("download image via %s attempt %d failed: %v", candidate, retry+1, err)
		}
		if ci < len(candidates)-1 {
			log.Warnf("image download failed via %s, trying next source", candidate)
		}
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no image sources available for %s", imageURL)
}

// BuildImageCandidates 按作品源 URL 生成图片下载候选: twitter 走 twimg 反代,
// 其余走 pixiv 代理链 (代理优先, 官方域名兜底)。公共逻辑见 imgcandidates 包。
func BuildImageCandidates(sourceURL, imageURL string) []string {
	return imgcandidates.ForURL(sourceURL, imageURL)
}

// cachedURLProxyMismatch 判断缓存的图片 URL 是否未使用当前配置的代理
// (twitter 看 twimg 反代; 其他按 pixiv img_proxy 判断), 用于下载前刷新。
func cachedURLProxyMismatch(sourceURL, original string) bool {
	return imgcandidates.ProxyMismatch(sourceURL, original)
}
