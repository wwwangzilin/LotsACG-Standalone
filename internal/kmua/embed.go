package kmua

import "embed"

// ProjectFS 内嵌的 kmua-bot (Python) 完整源码。
// 通过 exe 直接启动, 无需单独部署。
//
//go:embed project
var ProjectFS embed.FS
