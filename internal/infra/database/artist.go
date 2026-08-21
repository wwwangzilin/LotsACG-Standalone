package database

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/unvgo/ouid"
	"gorm.io/gorm"
)

func (d *DB) GetArtistByID(ctx context.Context, id ouid.OUID) (*entity.Artist, error) {
	artist, err := gorm.G[entity.Artist](d.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &artist, nil
}

func (d *DB) GetArtistByUID(ctx context.Context, uid string, source shared.SourceType) (*entity.Artist, error) {
	artist, err := gorm.G[entity.Artist](d.db).Where("uid = ? AND type = ?", uid, source).First(ctx)
	if err != nil {
		return nil, err
	}
	return &artist, nil
}

func (d *DB) UpdateArtist(ctx context.Context, patch *entity.Artist) error {
	_, err := gorm.G[entity.Artist](d.db).Where("id = ?", patch.ID).Updates(ctx, *patch)
	return err
}

func (d *DB) CreateArtist(ctx context.Context, artist *entity.Artist) (*ouid.OUID, error) {
	result := gorm.WithResult()
	err := gorm.G[entity.Artist](d.db, result).Create(ctx, artist)
	if err != nil {
		return nil, err
	}
	return &artist.ID, nil
}

func (d *DB) ListArtists(ctx context.Context, offset, limit int) ([]entity.Artist, error) {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var artists []entity.Artist
	err := d.db.WithContext(ctx).Model(&entity.Artist{}).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&artists).Error
	if err != nil {
		return nil, err
	}
	return artists, nil
}

func (d *DB) CountArtists(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&entity.Artist{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
