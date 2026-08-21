package repo

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/unvgo/ouid"
)

type Artist interface {
	GetArtistByID(ctx context.Context, id ouid.OUID) (*entity.Artist, error)
	GetArtistByUID(ctx context.Context, uid string, sourceType shared.SourceType) (*entity.Artist, error)
	UpdateArtist(ctx context.Context, patch *entity.Artist) error
	CreateArtist(ctx context.Context, artist *entity.Artist) (*ouid.OUID, error)
	// ListArtists 分页返回作者列表 (offset/limit, 按创建时间正序)。
	ListArtists(ctx context.Context, offset, limit int) ([]entity.Artist, error)
	// CountArtists 返回作者总数。
	CountArtists(ctx context.Context) (int64, error)
}
