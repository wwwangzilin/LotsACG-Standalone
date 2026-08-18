package infra

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage/local"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage/telegram"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage/webdav"
)

func initStorage(ctx context.Context) error {
	local.Init()
	telegram.Init()
	webdav.Init()

	return storage.InitAll(ctx)
}
