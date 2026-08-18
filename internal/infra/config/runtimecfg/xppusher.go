package runtimecfg

// XPPusherConfig 配置内置管理的 Pixiv-XP-Pusher (Python) 进程。
type XPPusherConfig struct {
	// Dir XP-Pusher 项目目录; 为空时使用 exe 内置 (内嵌源码自动提取到 <exe>/xppusher)
	Dir string `toml:"dir" mapstructure:"dir" json:"dir" yaml:"dir"`
	// Python Python 解释器路径; 为空时自动选择 venv/系统 python, 缺失时自动创建 venv 并安装依赖
	Python string `toml:"python" mapstructure:"python" json:"python" yaml:"python"`
	// Command 主脚本文件名, 默认 main.py
	Command string `toml:"command" mapstructure:"command" json:"command" yaml:"command"`
	// Args 附加参数 (空格分隔), 如 "--now"
	Args string `toml:"args" mapstructure:"args" json:"args" yaml:"args"`
	// Config 初始化 config.yaml 的来源文件 (外部已有配置, 例如 D:/projects/xp/Pixiv-XP-Pusher/config.yaml);
	// 为空时使用内置 config.example.yaml
	Config string `toml:"config" mapstructure:"config" json:"config" yaml:"config"`
	// AutoStart exe 启动时是否自动拉起 XP-Pusher
	AutoStart bool `toml:"auto_start" mapstructure:"auto_start" json:"auto_start" yaml:"auto_start"`
}
