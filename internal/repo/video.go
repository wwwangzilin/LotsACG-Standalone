package repo

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/unvgo/ouid"
)

type Video interface {
	UpdateVideoTelegramInfoByID(ctx context.Context, id ouid.OUID, tgInfo *shared.TelegramInfo) (*entity.Video, error)
}
