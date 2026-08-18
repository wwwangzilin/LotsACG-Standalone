package service

import (
	"context"
	"maps"
	"regexp"
	"strings"

	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
)

var urlRegex = regexp.MustCompile(`https?://[^\s<>"']+`)

func (s *Service) Source(sourceType shared.SourceType) source.ArtworkSource {
	return s.sources[sourceType]
}

func (s *Service) FindSourceURL(text string) string {
	urls := s.FindSourceURLs(text)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func (s *Service) FindSourceURLs(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	var urls []string
	seen := make(map[string]struct{})
	for _, raw := range urlRegex.FindAllString(text, -1) {
		candidate := strings.TrimSpace(strings.TrimRight(raw, " \t.,;:!?)]}"))
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		if s.isSupportedURL(candidate) {
			seen[candidate] = struct{}{}
			urls = append(urls, candidate)
		}
	}
	if len(urls) > 0 {
		return urls
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if s.isSupportedURL(trimmed) {
		return []string{trimmed}
	}
	return nil
}

func (s *Service) isSupportedURL(text string) bool {
	for _, sou := range s.sources {
		if _, ok := sou.MatchesSourceURL(text); ok {
			return true
		}
	}
	return false
}

func (s *Service) FetchArtworkInfo(ctx context.Context, sourceURL string) (*dto.FetchedArtwork, error) {
	for _, sou := range s.sources {
		if _, ok := sou.MatchesSourceURL(sourceURL); ok {
			return sou.GetArtworkInfo(ctx, sourceURL)
		}
	}
	return nil, oops.New("no supported source found")
}

// FetchNewArtworks fetches the latest artworks from all sources (e.g. Pixiv RSS).
func (s *Service) FetchNewArtworks(ctx context.Context, limit int) ([]*dto.FetchedArtwork, error) {
	artworks := make([]*dto.FetchedArtwork, 0)
	errs := make([]error, 0)
	for _, sou := range s.sources {
		fetched, err := sou.FetchNewArtworks(ctx, limit)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		artworks = append(artworks, fetched...)
	}
	if len(errs) > 0 {
		return artworks, oops.Join(errs...)
	}
	return artworks, nil
}

// FindArtistPageURL 返回文本中匹配的画师主页链接 (仅第一个)。
func (s *Service) FindArtistPageURL(text string) string {
	for _, sou := range s.sources {
		if lister, ok := sou.(source.ArtistArtworkLister); ok {
			if url, ok := lister.MatchArtistPageURL(text); ok {
				return url
			}
		}
	}
	return ""
}

// FetchArtistArtworks 返回指定画师主页下的全部作品完整链接 (limit<=0 表示全部)。
func (s *Service) FetchArtistArtworks(ctx context.Context, artistPageURL string, limit int) ([]string, error) {
	for _, sou := range s.sources {
		if lister, ok := sou.(source.ArtistArtworkLister); ok {
			if _, ok := lister.MatchArtistPageURL(artistPageURL); ok {
				return lister.FetchArtistArtworks(ctx, artistPageURL, limit)
			}
		}
	}
	return nil, oops.New("no artist page url matched")
}

// GenerateArtworkDescription 使用 AI 根据作品标签生成中文描述。
func (s *Service) GenerateArtworkDescription(ctx context.Context, artwork shared.ArtworkLike) (string, error) {
	if s.aiapi == nil || !s.aiapi.Enabled() {
		return "", oops.New("ai api not enabled")
	}
	return s.aiapi.GenerateDescription(ctx, artwork.GetTitle(), artwork.GetTags())
}

func (s *Service) PrettyFileName(artwork shared.ArtworkLike, picture shared.PictureLike) string {
	for _, sou := range s.sources {
		if _, ok := sou.MatchesSourceURL(artwork.GetSourceURL()); ok {
			return sou.PrettyFileName(artwork, picture)
		}
	}
	ext, _ := strutil.GetFileExtFromURL(picture.GetOriginal())
	return strings.ToLower(strutil.MD5Hash(picture.GetOriginal())) + ext
}

func (s *Service) Sources() map[shared.SourceType]source.ArtworkSource {
	return maps.Clone(s.sources)
}
