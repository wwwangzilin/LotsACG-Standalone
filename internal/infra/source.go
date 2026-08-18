package infra

import (
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/bilibili"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/danbooru"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/kemono"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/nhentai"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/pixiv"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/twitter"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/yandere"
)

func initSource() {
	nhentai.Init()
	pixiv.Init()
	twitter.Init()
	yandere.Init()
	bilibili.Init()
	danbooru.Init()
	kemono.Init()

	source.InitAll()
}
