package repo

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/unvgo/ouid"
)

type Ugoira interface {
	UpdateUgoiraTelegramInfoByID(ctx context.Context, id ouid.OUID, tgInfo *shared.TelegramInfo) (*entity.UgoiraMeta, error)
}
