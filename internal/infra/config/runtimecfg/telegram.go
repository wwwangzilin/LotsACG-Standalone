package runtimecfg

type TelegramConfig struct {
	BotToken        string  `toml:"bot_token" mapstructure:"bot_token" json:"bot_token" yaml:"bot_token"`
	APIURL          string  `toml:"api_url" mapstructure:"api_url" json:"api_url" yaml:"api_url"`
	Username        string  `toml:"username" mapstructure:"username" json:"username" yaml:"username"`
	CaptionTemplate string  `toml:"caption_template" mapstructure:"caption_template" json:"caption_template" yaml:"caption_template"`
	Proxy           string  `toml:"proxy" mapstructure:"proxy" json:"proxy" yaml:"proxy"`
	Admins          []int64 `toml:"admins" mapstructure:"admins" json:"admins" yaml:"admins"`
	// AllowedUsers 配置中登记的用户白名单。仅当非空时启用白名单模式:
	// 未登记的用户(游客)只允许 /start /help 和 /files(获取原图)。
	AllowedUsers []int64                     `toml:"allowed_users" mapstructure:"allowed_users" json:"allowed_users" yaml:"allowed_users"`
	ExtraTarget  []TelegramExtraTargetConfig `toml:"extra_target" mapstructure:"extra_target" json:"extra_target" yaml:"extra_target"`
	// SendChannels 除主频道外的额外发送频道 (配置文件方式), 首次启动时导入 KV。
	// 之后可用 /channel 命令管理; 每个频道可配置发布规则 (R18/标签/画师/仅链接)。
	SendChannels []TelegramSendChannelConfig `toml:"send_channels" mapstructure:"send_channels" json:"send_channels" yaml:"send_channels"`
	Retry        BotRetryConfig              `toml:"retry" mapstructure:"retry" json:"retry" yaml:"retry"`
	// Channel  bool    `toml:"channel" mapstructure:"channel" json:"channel" yaml:"channel"`
	ChatID  int64 `toml:"chat_id" mapstructure:"chat_id" json:"chat_id" yaml:"chat_id"`
	GroupID int64 `toml:"group_id" mapstructure:"group_id" json:"group_id" yaml:"group_id"`
	Disable bool  `toml:"disable" mapstructure:"disable" json:"disable" yaml:"disable"`
}

// 额外的发送的目标聊天, 将会在作品信息的操作键盘上显示发送到这些频道(但不落库)
type TelegramExtraTargetConfig struct {
	Title  string `toml:"title" mapstructure:"title" json:"title" yaml:"title"` // 在按钮上显示的标题
	ChatID int64  `toml:"chat_id" mapstructure:"chat_id" json:"chat_id" yaml:"chat_id"`
}

// TelegramSendChannelConfig 是配置文件方式定义的发送频道。
// 与 /channel 命令添加的频道等价, 首次启动时导入 KV; enabled 缺省为 true。
type TelegramSendChannelConfig struct {
	ChatID         int64    `toml:"chat_id" mapstructure:"chat_id" json:"chat_id" yaml:"chat_id"`
	Title          string   `toml:"title" mapstructure:"title" json:"title" yaml:"title"`
	Enabled        *bool    `toml:"enabled" mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	R18Mode        string   `toml:"r18_mode" mapstructure:"r18_mode" json:"r18_mode" yaml:"r18_mode"` // follow|allow|deny|only
	LinkOnly       bool     `toml:"link_only" mapstructure:"link_only" json:"link_only" yaml:"link_only"`
	IncludeTags    []string `toml:"include_tags" mapstructure:"include_tags" json:"include_tags" yaml:"include_tags"`
	ExcludeTags    []string `toml:"exclude_tags" mapstructure:"exclude_tags" json:"exclude_tags" yaml:"exclude_tags"`
	IncludeArtists []string `toml:"include_artists" mapstructure:"include_artists" json:"include_artists" yaml:"include_artists"`
	ExcludeArtists []string `toml:"exclude_artists" mapstructure:"exclude_artists" json:"exclude_artists" yaml:"exclude_artists"`
}

type BotRetryConfig struct {
	MaxAttempts  int     `toml:"max_attempts" mapstructure:"max_attempts" json:"max_attempts" yaml:"max_attempts"`
	ExponentBase float64 `toml:"exponent_base" mapstructure:"exponent_base" json:"exponent_base" yaml:"exponent_base"`
	StartDelay   int64   `toml:"start_delay" mapstructure:"start_delay" json:"start_delay" yaml:"start_delay"`
	MaxDelay     int64   `toml:"max_delay" mapstructure:"max_delay" json:"max_delay" yaml:"max_delay"`
}
