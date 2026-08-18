package pixiv

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// SearchArtworksByTags 通过 Pixiv 搜索接口按 tag 搜索最新作品。
// 搜索多个 tag 会按空格组合 (AND 匹配, 与 XP-Pusher 一致)。返回轻量条目
// (含 title/tags/封面/收藏数), 便于快速打分; 完整图片信息可再调 GetArtworkInfo。
func (p *Pixiv) SearchArtworksByTags(ctx context.Context, tags []string, limit int) ([]*dto.FetchedArtwork, error) {
	return p.searchArtworksByTags(ctx, tags, limit, "date_d", "all")
}

// SearchArtworksByTagsOrdered 与 SearchArtworksByTags 相同, 但允许指定排序方式。
// order: date_d(最新) / popular_desc(热门, 收藏降序)。
func (p *Pixiv) SearchArtworksByTagsOrdered(ctx context.Context, tags []string, limit int, order string) ([]*dto.FetchedArtwork, error) {
	return p.searchArtworksByTags(ctx, tags, limit, order, "all")
}

// SearchArtworksByTagsOrderedWithMode 支持指定 R18 过滤模式。
// r18Mode: all(全部) / safe(全年龄) / r18(仅 R18)。
func (p *Pixiv) SearchArtworksByTagsOrderedWithMode(ctx context.Context, tags []string, limit int, order, r18Mode string) ([]*dto.FetchedArtwork, error) {
	return p.searchArtworksByTags(ctx, tags, limit, order, r18Mode)
}

func (p *Pixiv) searchArtworksByTags(ctx context.Context, tags []string, limit int, order, r18Mode string) ([]*dto.FetchedArtwork, error) {
	if len(tags) == 0 {
		return nil, oops.New("no tags provided for pixiv search")
	}
	if limit <= 0 {
		limit = 20
	}
	if order == "" {
		order = "date_d"
	}
	if r18Mode == "" {
		r18Mode = "all"
	}
	// Pixiv 搜索关键词: 多个 tag 用空格分隔 (AND 语义)
	keyword := strings.Join(tags, " ")

	// 尝试指定排序, 失败时回退到 date_d
	orders := []string{order, "date_d"}
	var lastErr error
	for oi, curOrder := range orders {
		searchURL := "https://www.pixiv.net/ajax/search/artworks/" + url.PathEscape(keyword) +
			"?word=" + url.QueryEscape(keyword) +
			"&order=" + url.QueryEscape(curOrder) +
			"&mode=" + url.QueryEscape(r18Mode) + "&p=1&s_mode=s_tag&type=all&lang=zh"
		artworks, err := p.searchURL(ctx, searchURL, limit)
		if err == nil {
			if len(artworks) > 0 || oi == len(orders)-1 {
				return artworks, nil
			}
			// 空结果但还有回退排序可试
			lastErr = nil
			continue
		}
		lastErr = err
		log.Warnf("pixiv tag search failed with order %s, fallback to date_d", curOrder)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// searchURL 执行一次 Pixiv ajax 搜索请求并解析。
func (p *Pixiv) searchURL(ctx context.Context, searchURL string, limit int) ([]*dto.FetchedArtwork, error) {
	var lastErr error
	for i := 0; i < len(p.reqClients); i++ {
		client := p.nextClient()
		resp, err := client.R().
			SetContext(ctx).
			SetHeader("Referer", "https://www.pixiv.net/").
			SetHeader("Accept", "application/json").
			Get(searchURL)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv tag search request failed with account %d: %v", i+1, err)
			continue
		}
		body, err := respBodyBytes(resp)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv tag search decompress failed with account %d: %v", i+1, err)
			continue
		}
		var searchResp PixivSearchResp
		if err := json.Unmarshal(body, &searchResp); err != nil {
			preview := body
			if len(preview) > 200 {
				preview = preview[:200]
			}
			lastErr = oops.Wrapf(err, "unmarshal pixiv search response")
			log.Warnf("pixiv tag search unmarshal failed with account %d: %v; body=%q", i+1, err, string(preview))
			continue
		}
		if searchResp.Error {
			lastErr = oops.Errorf("pixiv search response error: %s", searchResp.Message)
			log.Warnf("pixiv tag search returned error with account %d: %s", i+1, searchResp.Message)
			continue
		}
		if searchResp.Body == nil || searchResp.Body.IllustManga == nil || len(searchResp.Body.IllustManga.Data) == 0 {
			return nil, nil
		}
		artworks := make([]*dto.FetchedArtwork, 0, limit)
		for _, item := range searchResp.Body.IllustManga.Data {
			if len(artworks) >= limit {
				break
			}
			// 只取普通插画(illustType 0), 跳过动图
			if item.IllustType != 0 {
				continue
			}
			fetched := searchIllustToFetched(item, p.cfg.ImgProxy)
			if fetched == nil {
				continue
			}
			artworks = append(artworks, fetched)
		}
		if len(artworks) > 0 {
			return artworks, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, oops.New("no pixiv accounts available")
}

// searchIllustToFetched 将搜索结果中的单个插画条目转换为轻量 FetchedArtwork。
// 仅包含 title/tags/封面缩略图, 用于快速打分排序。
func searchIllustToFetched(item *PixivSearchIllustData, imgProxy string) *dto.FetchedArtwork {
	if item == nil || item.ID == "" || item.URL == "" {
		return nil
	}
	original := strings.Replace(item.URL, "i.pximg.net", imgProxy, 1)
	tags := make([]string, 0, len(item.Tags))
	for _, t := range item.Tags {
		if t != "" {
			tags = append(tags, t)
		}
	}
	fetched := &dto.FetchedArtwork{
		Title:         item.Title,
		Description:   item.Description,
		R18:           item.XRestrict != 0,
		SourceType:    shared.SourceTypePixiv,
		SourceURL:     fmt.Sprintf("https://www.pixiv.net/artworks/%s", item.ID),
		Tags:          tags,
		BookmarkCount: item.BookmarkCount,
		ViewCount:     item.ViewCount,
		CreateDate:    item.CreateDate,
		Pictures: []*dto.FetchedPicture{
			{
				Index:     0,
				Thumbnail: original,
				Original:  original,
			},
		},
	}
	if item.UserID != "" {
		fetched.Artist = &dto.FetchedArtist{
			Name:     item.UserName,
			Type:     shared.SourceTypePixiv,
			UID:      item.UserID,
			Username: item.UserAccount,
		}
	}
	return fetched
}
