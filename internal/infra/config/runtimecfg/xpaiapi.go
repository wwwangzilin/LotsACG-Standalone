package runtimecfg

// XPAIAPIConfig 配置 XP 画像 AI API (OpenAI 兼容), 用于 XP 画像构建与推荐。
// 结构参考 Pixiv-XP-Pusher 的 profiler.ai 配置。
type XPAIAPIConfig struct {
	// Enabled 是否启用 XP AI 画像
	Enabled bool `toml:"enabled" mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// Provider 提供商: openai / local
	Provider string `toml:"provider" mapstructure:"provider" json:"provider" yaml:"provider"`
	// APIKey API 密钥
	APIKey string `toml:"api_key" mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	// BaseURL API 地址 (OpenAI 兼容)
	BaseURL string `toml:"base_url" mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	// Model 使用的模型
	Model string `toml:"model" mapstructure:"model" json:"model" yaml:"model"`
	// Embedding 文本向量配置 (语义匹配)
	Embedding XPEmbeddingConfig `toml:"embedding" mapstructure:"embedding" json:"embedding" yaml:"embedding"`
	// ScanLimit 构建画像时扫描的作品数量上限
	ScanLimit int `toml:"scan_limit" mapstructure:"scan_limit" json:"scan_limit" yaml:"scan_limit"`
	// DiscoveryRate 探索率 (0~1): 推荐中随机探索新风格的比例
	DiscoveryRate float64 `toml:"discovery_rate" mapstructure:"discovery_rate" json:"discovery_rate" yaml:"discovery_rate"`
	// AutoTag 是否在新作品入库时使用 AI 自动生成/补充标签
	AutoTag bool `toml:"auto_tag" mapstructure:"auto_tag" json:"auto_tag" yaml:"auto_tag"`
	// Pixiv Pixiv 账号配置 (用于访问收藏夹构建画像)
	Pixiv XPPixivConfig `toml:"pixiv" mapstructure:"pixiv" json:"pixiv" yaml:"pixiv"`
}

// XPPixivConfig Pixiv 账号配置, 用于通过 OAuth 访问收藏夹构建 XP 画像。
type XPPixivConfig struct {
	// RefreshToken Pixiv refresh_token (在 https://oauth.secure.pixiv.net 获取)
	RefreshToken string `toml:"refresh_token" mapstructure:"refresh_token" json:"refresh_token" yaml:"refresh_token"`
	// UserID Pixiv 用户 ID (用于获取该用户的收藏)
	UserID string `toml:"user_id" mapstructure:"user_id" json:"user_id" yaml:"user_id"`
}

// XPEmbeddingConfig 文本向量 (Embedding) 配置。
type XPEmbeddingConfig struct {
	// Model 向量模型名
	Model string `toml:"model" mapstructure:"model" json:"model" yaml:"model"`
	// Dimensions 向量维度
	Dimensions int `toml:"dimensions" mapstructure:"dimensions" json:"dimensions" yaml:"dimensions"`
}
