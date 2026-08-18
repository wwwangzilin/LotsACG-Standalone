package runtimecfg

// AIAPIConfig 配置 AI API (OpenAI 兼容) 用于推荐系统中自动关联相似 tag。
type AIAPIConfig struct {
	Enable  bool   `toml:"enable" mapstructure:"enable" json:"enable" yaml:"enable"`
	BaseURL string `toml:"base_url" mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	APIKey  string `toml:"api_key" mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	Model   string `toml:"model" mapstructure:"model" json:"model" yaml:"model"`

	// RecommendTags 一次请求最多生成的关联 tag 数量
	RecommendTags int `toml:"recommend_tags" mapstructure:"recommend_tags" json:"recommend_tags" yaml:"recommend_tags"`

	// AutoTag 是否在新作品入库时使用 AI 自动生成/补充标签
	AutoTag bool `toml:"auto_tag" mapstructure:"auto_tag" json:"auto_tag" yaml:"auto_tag"`
}
