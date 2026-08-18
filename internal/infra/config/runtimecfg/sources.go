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
	FxTwitterDomain string `toml:"fx_twitter_domain" mapstructure:"fx_twitter_domain" json:"fx_twitter_domain" yaml:"fx_twitter_domain"`
	Disable         bool   `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
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
