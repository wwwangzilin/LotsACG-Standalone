package repo

import (
	"context"

	"github.com/unvgo/ouid"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
)

type CachedArtwork interface {
	CreateCachedArtwork(ctx context.Context, cachedArt *entity.CachedArtwork) (*entity.CachedArtwork, error)
	ResetPostingCachedArtworkStatus(ctx context.Context) error
	DeleteCachedArtworkByID(ctx context.Context, id ouid.OUID) error
	GetCachedArtworkByURL(ctx context.Context, url string) (*entity.CachedArtwork, error)
	SaveCachedArtwork(ctx context.Context, artwork *entity.CachedArtwork) (*entity.CachedArtwork, error)
	CountCachedArtwork(ctx context.Context) (int64, error)
}
