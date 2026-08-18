package service

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/unvgo/ouid"
)

func (s *Service) GetArtistByID(ctx context.Context, id ouid.OUID) (*entity.Artist, error) {
	return s.repos.Artist().GetArtistByID(ctx, id)
}
