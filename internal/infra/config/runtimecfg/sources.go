package runtimecfg

import "strings"

type SourceConfig struct {
	Twitter  SourceTwitterConfig  `toml:"twitter" mapstructure:"twitter" json:"twitter" yaml:"twitter"`
	Proxy    string               `toml:"proxy" mapstructure:"proxy" json:"proxy" yaml:"proxy"`
	Pixiv    SourcePixivConfig    `toml:"pixiv" mapstructure:"pixiv" json:"pixiv" yaml:"pixiv"`
	Bilibili SourceBilibiliConfig `toml:"bilibili" mapstructure:"bilibili" json:"bilibili" yaml:"bilibili"`
	Danbooru SourceDanbooruConfig `toml:"danbooru" mapstructure:"danbooru" json:"danbooru" yaml:"danbooru"`
	Kemono   SourceKemonoConfig   `toml:"kemono" mapstructure:"kemono" json:"kemono" yaml:"kemono"`
	Yandere  SourceYandereConfig  `toml:"yandere" mapstructure:"yandere" json:"yandere" yaml:"yandere"`
	Nhentai  SourceNhentaiConfig  `toml:"nhentai" mapstructure:"nhentai" json:"nhentai" yaml:"nhentai"`
	Misskey  SourceMisskeyConfig  `toml:"misskey" mapstructure:"misskey" json:"misskey" yaml:"misskey"`
}

type SourcePixivConfig struct {
	ImgProxy  string             `toml:"img_proxy" mapstructure:"img_proxy" json:"img_proxy" yaml:"img_proxy"`
	ImgProxies []string          `toml:"img_proxies" mapstructure:"img_proxies" json:"img_proxies" yaml:"img_proxies"`
	RssURLs   []string           `toml:"rss_urls" mapstructure:"rss_urls" json:"rss_urls" yaml:"rss_urls"`
	Cookies   []CookieConfig     `toml:"cookies" mapstructure:"cookies" json:"cookies" yaml:"cookies"`
	Accounts  []PixivAccountConfig `toml:"accounts" mapstructure:"accounts" json:"accounts" yaml:"accounts"`
	Disable   bool               `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

// ImgProxyHosts returns the ordered list of image proxy hosts to try when downloading,
// primary first, then fallbacks. Empty values are skipped.
func (c SourcePixivConfig) ImgProxyHosts() []string {
	seen := make(map[string]struct{}, 4)
	hosts := make([]string, 0, 4)
	add := func(h string) {
		h = strings.TrimSpace(h)
		if h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		hosts = append(hosts, h)
	}
	add(c.ImgProxy)
	for _, p := range c.ImgProxies {
		add(p)
	}
	return hosts
}

type PixivAccountConfig struct {
	Name    string         `toml:"name" mapstructure:"name" json:"name" yaml:"name"`
	Cookies []CookieConfig `toml:"cookies" mapstructure:"cookies" json:"cookies" yaml:"cookies"`
}

type SourceTwitterConfig struct {
	FxTwitterDomain string   `toml:"fx_twitter_domain" mapstructure:"fx_twitter_domain" json:"fx_twitter_domain" yaml:"fx_twitter_domain"`
	ImgProxy        string   `toml:"img_proxy" mapstructure:"img_proxy" json:"img_proxy" yaml:"img_proxy"`             // 图片反代 (pbs.twimg.com → 反代域名, 空=不重写)
	ImgProxies      []string `toml:"img_proxies" mapstructure:"img_proxies" json:"img_proxies" yaml:"img_proxies"`     // 备用反代, 按顺序降级
	Disable         bool     `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

// ImgProxyHosts returns the ordered list of Twitter image proxy hosts to try
// when downloading (primary first, then fallbacks). Empty values are skipped.
func (c SourceTwitterConfig) ImgProxyHosts() []string {
	seen := make(map[string]struct{}, 4)
	hosts := make([]string, 0, 4)
	add := func(h string) {
		h = strings.TrimSpace(h)
		if h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		hosts = append(hosts, h)
	}
	add(c.ImgProxy)
	for _, p := range c.ImgProxies {
		add(p)
	}
	return hosts
}

type SourceBilibiliConfig struct {
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

type SourceDanbooruConfig struct {
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

type SourceKemonoConfig struct {
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

type SourceYandereConfig struct {
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

type SourceNhentaiConfig struct {
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

// SourceMisskeyConfig 配置 Misskey (联邦宇宙) 数据源。
type SourceMisskeyConfig struct {
	// Disable 是否禁用该源
	Disable bool `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
	// Instance Misskey 实例地址, 如 "https://misskey.io" 或 "https://mk.example.com"
	Instance string `toml:"instance" mapstructure:"instance" json:"instance" yaml:"instance"`
	// UserID 关注的用户 ID (可选), 为空时拉取全局时间线; 可填 "userID1,userID2" 多个
	UserID string `toml:"user_id" mapstructure:"user_id" json:"user_id" yaml:"user_id"`
	// Limit 每次拉取数量上限 (0=默认 20)
	Limit int `toml:"limit" mapstructure:"limit" json:"limit" yaml:"limit"`
}
