package runtimecfg

// KMuaConfig 配置内置管理的 kmua-bot (Telegram 群聊机器人, Python) 进程。
// kmua-bot 源码内嵌进 exe (internal/kmua/project), 首次启动自动提取 + uv 安装依赖。
type KMuaConfig struct {
	// Dir kmua-bot 项目目录; 为空时使用 exe 内置 (内嵌源码自动提取到 <exe>/kmua)
	Dir string `toml:"dir" mapstructure:"dir" json:"dir" yaml:"dir"`
	// UV uv 可执行文件路径 (用于创建 venv 并安装依赖); 为空时在 PATH 中查找
	UV string `toml:"uv" mapstructure:"uv" json:"uv" yaml:"uv"`
	// Python Python 解释器路径; 为空时用 uv 管理 (uv 自动下载 Python 3.13 并建 .venv)
	Python string `toml:"python" mapstructure:"python" json:"python" yaml:"python"`
	// Command 启动参数, 默认 "-m kmua"
	Command string `toml:"command" mapstructure:"command" json:"command" yaml:"command"`
	// Args 附加参数 (空格分隔)
	Args string `toml:"args" mapstructure:"args" json:"args" yaml:"args"`
	// AutoStart exe 启动时是否自动拉起 kmua-bot
	AutoStart bool `toml:"auto_start" mapstructure:"auto_start" json:"auto_start" yaml:"auto_start"`
	// Token kmua-bot 的 bot token (首次生成 settings.toml 时写入)
	Token string `toml:"token" mapstructure:"token" json:"token" yaml:"token"`
	// Owners kmua-bot 的 owner IDs (首次生成 settings.toml 时写入)
	Owners []int64 `toml:"owners" mapstructure:"owners" json:"owners" yaml:"owners"`
	// Proxy Telegram 连接代理 (GFW 环境需要), 如 http://127.0.0.1:7899; 写入 settings.toml
	Proxy string `toml:"proxy" mapstructure:"proxy" json:"proxy" yaml:"proxy"`
	// Webapp 是否启用 kmua 管理面板 (Telegram Mini App)
	Webapp bool `toml:"webapp" mapstructure:"webapp" json:"webapp" yaml:"webapp"`
	// WebappPort 面板监听端口, 默认 8180
	WebappPort int `toml:"webapp_port" mapstructure:"webapp_port" json:"webapp_port" yaml:"webapp_port"`
	// WebappURL 面板公网 HTTPS 地址 (Telegram 不会打开 HTTP 的 Mini App)
	WebappURL string `toml:"webapp_url" mapstructure:"webapp_url" json:"webapp_url" yaml:"webapp_url"`
	// WebappShortName 在 BotFather 注册的 Mini App short name
	WebappShortName string `toml:"webapp_short_name" mapstructure:"webapp_short_name" json:"webapp_short_name" yaml:"webapp_short_name"`
	// Settings 初始 settings.toml 来源文件 (外部已有配置); 为空时用内嵌模板 + 上面字段生成
	Settings string `toml:"settings" mapstructure:"settings" json:"settings" yaml:"settings"`
}
