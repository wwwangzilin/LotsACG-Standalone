package service

import (
	"context"

	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
)

// ConvertFetchedToCached 将 FetchedArtwork 转换为可用于展示/发布的 CachedArtworkData。
func ConvertFetchedToCached(fetched *dto.FetchedArtwork) (*entity.CachedArtworkData, error) {
	if fetched == nil {
		return nil, oops.New("nil fetched artwork")
	}
	cached := &entity.CachedArtworkData{
		ID:          fetched.SourceURL,
		Title:       fetched.Title,
		Description: fetched.Description,
		R18:         fetched.R18,
		Tags:        fetched.Tags,
		SourceURL:   fetched.SourceURL,
		SourceType:  fetched.SourceType,
	}
	if fetched.Artist != nil {
		cached.Artist = &entity.CachedArtist{
			Name:     fetched.Artist.Name,
			UID:      fetched.Artist.UID,
			Username: fetched.Artist.Username,
		}
	}
	for _, p := range fetched.Pictures {
		cached.Pictures = append(cached.Pictures, &entity.CachedPicture{
			OrderIndex: p.Index,
			Thumbnail:  p.Thumbnail,
			Original:   p.Original,
			Width:      p.Width,
			Height:     p.Height,
		})
	}
	for _, u := range fetched.UgoiraMetas {
		cached.UgoiraMetas = append(cached.UgoiraMetas, &entity.CachedUgoiraMeta{
			OrderIndex: u.Index,
			MetaData:   u.Data,
		})
	}
	for _, v := range fetched.Videos {
		cached.Videos = append(cached.Videos, &entity.CachedVideo{
			OrderIndex: v.Index,
			URL:        v.URL,
			Width:      v.Width,
			Height:     v.Height,
			Duration:   v.Duration,
			Poster:     v.Poster,
			MimeType:   v.MimeType,
		})
	}
	return cached, nil
}

// SearchNewArtworksByTagsOrdered 按 tag 搜索, 支持排序 (date_d / popular_desc)。
func (s *Service) SearchNewArtworksByTagsOrdered(ctx context.Context, tags []string, limit int, order string) ([]*dto.FetchedArtwork, error) {
	return s.SearchNewArtworksByTagsOrderedWithMode(ctx, tags, limit, order, "all", "")
}

// SearchNewArtworksByTagsOrderedWithMode 按 tag 搜索并支持 R18 过滤模式与图源限定。
// r18Mode: all(全部) / safe(全年龄) / r18(仅 R18)。
// sourceType: 仅从指定图源搜索 (如 "pixiv"), 空字符串表示全部图源。
func (s *Service) SearchNewArtworksByTagsOrderedWithMode(ctx context.Context, tags []string, limit int, order, r18Mode, sourceType string) ([]*dto.FetchedArtwork, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if r18Mode == "" {
		r18Mode = "all"
	}
	artworks := make([]*dto.FetchedArtwork, 0)
	errs := make([]error, 0)
	seen := make(map[string]struct{})
	for st, sou := range s.sources {
		if sourceType != "" && string(st) != sourceType {
			continue
		}
		// 优先使用支持 R18 模式过滤的源
		if searcher, ok := sou.(source.ArtworkTagSearcherOrderedWithMode); ok {
			fetched, err := searcher.SearchArtworksByTagsOrderedWithMode(ctx, tags, limit, order, r18Mode)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, art := range fetched {
				if art == nil || art.SourceURL == "" {
					continue
				}
				if _, ok := seen[art.SourceURL]; ok {
					continue
				}
				seen[art.SourceURL] = struct{}{}
				artworks = append(artworks, art)
			}
			continue
		}
		searcher, ok := sou.(source.ArtworkTagSearcherOrdered)
		if !ok {
			continue
		}
		fetched, err := searcher.SearchArtworksByTagsOrdered(ctx, tags, limit, order)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, art := range fetched {
			if art == nil || art.SourceURL == "" {
				continue
			}
			if _, ok := seen[art.SourceURL]; ok {
				continue
			}
			seen[art.SourceURL] = struct{}{}
			artworks = append(artworks, art)
		}
	}
	if len(errs) > 0 {
		return artworks, oops.Join(errs...)
	}
	return artworks, nil
}
