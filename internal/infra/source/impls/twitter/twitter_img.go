package twitter

import (
	"net/url"
	"strings"
)

// twitterImgDomain 是 Twitter 官方图片 CDN 域名, 直连被墙, 可配置反代域名替换。
const twitterImgDomain = "pbs.twimg.com"

// BuildTwitterImageCandidates returns the image URLs to try in order:
// configured proxies first (primary then fallbacks), and finally the official
// pbs.twimg.com domain. Non-twimg URLs are returned as-is (no rewrite).
// Deduplicates hosts.
func BuildTwitterImageCandidates(imageURL string, proxyHosts []string) []string {
	u, err := url.Parse(imageURL)
	if err != nil || u.Host == "" {
		return []string{imageURL}
	}
	// 只对 pbs.twimg.com 图片做反代重写, 其他域名原样返回
	if !isTwimgHost(u.Host) {
		return []string{imageURL}
	}
	originalHost := u.Host
	seen := make(map[string]struct{}, len(proxyHosts)+2)
	candidates := make([]string, 0, len(proxyHosts)+2)
	add := func(host string) {
		host = strings.TrimSpace(host)
		if host == "" {
			return
		}
		if _, ok := seen[host]; ok {
			return
		}
		seen[host] = struct{}{}
		dup := *u
		dup.Host = host
		candidates = append(candidates, dup.String())
	}
	// 配置的反代优先
	for _, h := range proxyHosts {
		add(h)
	}
	// 官方源兜底
	add(twitterImgDomain)
	// 若原始 host 不在候选中(例如子域名), 也补上
	add(originalHost)
	return candidates
}

// isTwimgHost 判断 host 是否为 pbs.twimg.com 或其子域名。
func isTwimgHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == twitterImgDomain || strings.HasSuffix(h, "."+twitterImgDomain)
}
