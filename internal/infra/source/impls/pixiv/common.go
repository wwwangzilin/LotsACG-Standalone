package pixiv

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	sourceReg             = regexp.MustCompile(`pixiv\.net/(?:en/)?(?:artworks/|i/|member_illust\.php\?(?:[\w=&]*\&|)illust_id=)(\d+)`)
	ErrUnmarshalPixivAjax = errors.New("error decoding artwork info, maybe the artwork is deleted")
)

const (
	pixivImgDomain = "i.pximg.net"
)

// BuildPixivImageCandidates returns the image URLs to try in order:
// configured proxies first (primary then fallbacks), and finally the official
// i.pximg.net domain. Deduplicates hosts.
func BuildPixivImageCandidates(imageURL string, proxyHosts []string) []string {
	u, err := url.Parse(imageURL)
	if err != nil || u.Host == "" {
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
	// 配置的代理优先 (manyacg -> pixiv.cat -> i.muxmus.com)
	for _, h := range proxyHosts {
		add(h)
	}
	// 官方源兜底
	add(pixivImgDomain)
	// 若原始 host 不在候选中(例如已经是官方或自定义域名), 也补上
	add(originalHost)
	return candidates
}
