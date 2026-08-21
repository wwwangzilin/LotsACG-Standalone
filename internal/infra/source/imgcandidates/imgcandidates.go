// Package imgcandidates 按作品来源生成图片下载候选 URL (代理优先, 官方兜底)。
package imgcandidates

import (
	"strings"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/pixiv"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/twitter"
)

// ForURL 按作品源 URL 生成图片下载候选:
//   - X/Twitter 作品 → twimg 反代 (配置 img_proxy) 优先, pbs.twimg.com 官方兜底
//   - 其他作品 → pixiv 代理链 (img_proxy/img_proxies) 优先, i.pximg.net 官方兜底
func ForURL(sourceURL, imageURL string) []string {
	cfg := runtimecfg.Get()
	if IsTwitterArtworkURL(sourceURL) {
		return twitter.BuildTwitterImageCandidates(imageURL, cfg.Source.Twitter.ImgProxyHosts())
	}
	return pixiv.BuildPixivImageCandidates(imageURL, cfg.Source.Pixiv.ImgProxyHosts())
}

// ProxyMismatch 判断缓存的图片 URL 是否未使用当前配置的代理, 用于下载前刷新:
//   - twitter 配置了反代但缓存仍是官方 twimg 域名
//   - pixiv 缓存 URL 不含当前 img_proxy
func ProxyMismatch(sourceURL, original string) bool {
	if original == "" {
		return false
	}
	cfg := runtimecfg.Get()
	if IsTwitterArtworkURL(sourceURL) {
		hosts := cfg.Source.Twitter.ImgProxyHosts()
		if len(hosts) == 0 {
			return false
		}
		return strings.Contains(original, "twimg.com") && !strings.Contains(strings.ToLower(original), strings.ToLower(hosts[0]))
	}
	imgProxy := cfg.Source.Pixiv.ImgProxy
	return imgProxy != "" && !strings.Contains(original, imgProxy)
}

// IsTwitterArtworkURL 粗略判断作品链接是否来自 X/Twitter。
func IsTwitterArtworkURL(sourceURL string) bool {
	u := strings.ToLower(sourceURL)
	return strings.Contains(u, "x.com/") || strings.Contains(u, "twitter.com/")
}
