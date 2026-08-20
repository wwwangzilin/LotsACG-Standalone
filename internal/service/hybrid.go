package service

import (
	"context"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/query"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

// HybridSearchResult 混合检索结果: 相似图片的作品 + 关键词建议。
type HybridSearchResult struct {
	Artworks    []*entity.Artwork `json:"artworks"`
	TagSuggest  []string          `json:"tag_suggest"`  // 从相似作品聚合的热门标签 (以图搜文的关键词建议)
	SearchTerms []string          `json:"search_terms"` // 关键词候选 (可继续用于文本搜索)
}

// SearchByImageHybrid 以图搜文: 用图片找到相似作品, 并聚合其标签作为关键词建议。
func (s *Service) SearchByImageHybrid(ctx context.Context, imageBytes []byte, limit int) (*HybridSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	hits, err := s.SearchPicturesByImage(ctx, imageBytes, 10, limit)
	if err != nil {
		return nil, err
	}

	result := &HybridSearchResult{
		Artworks: make([]*entity.Artwork, 0, len(hits)),
	}
	artworkSeen := make(map[string]struct{})
	tagFreq := make(map[string]int)

	for _, hit := range hits {
		pic := hit.Picture
		if pic == nil {
			continue
		}
		aw := pic.Artwork
		if aw == nil {
			continue
		}
		awID := aw.ID.Hex()
		if _, ok := artworkSeen[awID]; ok {
			continue
		}
		artworkSeen[awID] = struct{}{}
		result.Artworks = append(result.Artworks, aw)
		for _, tag := range aw.Tags {
			if tag == nil {
				continue
			}
			t := strings.TrimSpace(tag.Name)
			if t != "" {
				tagFreq[t]++
			}
		}
	}

	// 按频率排序标签
	type tagItem struct {
		tag string
		n   int
	}
	items := make([]tagItem, 0, len(tagFreq))
	for t, n := range tagFreq {
		items = append(items, tagItem{t, n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].n != items[j].n {
			return items[i].n > items[j].n
		}
		return items[i].tag < items[j].tag
	})
	maxSuggest := 12
	if len(items) < maxSuggest {
		maxSuggest = len(items)
	}
	for _, it := range items[:maxSuggest] {
		result.TagSuggest = append(result.TagSuggest, it.tag)
		result.SearchTerms = append(result.SearchTerms, it.tag)
	}

	return result, nil
}

// HybridSearchRequest 混合搜索请求 (Web/Telegram 共用)。
type HybridSearchRequest struct {
	ImageURL string `json:"image_url"` // 图片地址 (以图搜文)
	Query    string `json:"query"`     // 关键词 (文本搜索)
	Limit    int    `json:"limit"`
}

// HybridSearch 统一混合搜索入口: 纯文本 / 纯图片 / 图文结合。
// 文本搜索使用 meilisearch (query.ArtworkSearch.Hybrid 开启时由 searcher 做视觉重排)。
func (s *Service) HybridSearch(ctx context.Context, req *HybridSearchRequest) (*HybridSearchResult, error) {
	// 纯文本搜索
	if req.Query != "" && req.ImageURL == "" {
		artworks, err := s.SearchArtworks(ctx, &query.ArtworkSearch{
			Query:  req.Query,
			Hybrid: true, // 开启混合: 文本召回 + 视觉重排 (searcher 支持时)
			R18:    shared.R18TypeAll,
			Paginate: query.Paginate{
				Limit: limitOr(req.Limit, 20),
			},
		})
		if err != nil {
			return nil, err
		}
		return &HybridSearchResult{Artworks: artworks}, nil
	}
	// 图片为主: 以图搜文
	if req.ImageURL != "" {
		bytes, err := s.fetchImageBytes(ctx, req.ImageURL)
		if err != nil {
			return nil, err
		}
		return s.SearchByImageHybrid(ctx, bytes, req.Limit)
	}
	return nil, nil
}

// fetchImageBytes 下载图片字节 (带 Referer/UA 头, 适配 pixiv 等图床)。
func (s *Service) fetchImageBytes(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) LotsACG")
	req.Header.Set("Referer", "https://www.pixiv.net/")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &hybridHTTPError{Code: resp.StatusCode}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
}

type hybridHTTPError struct{ Code int }

func (e *hybridHTTPError) Error() string { return "hybrid fetch http error: " + strconv.Itoa(e.Code) }

func limitOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
