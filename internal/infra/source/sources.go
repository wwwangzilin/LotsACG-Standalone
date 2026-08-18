package source

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

type ArtworkSource interface {
	GetArtworkInfo(ctx context.Context, sourceUrl string) (*dto.FetchedArtwork, error)
	MatchesSourceURL(sourceUrl string) (string, bool)
	FetchNewArtworks(ctx context.Context, limit int) ([]*dto.FetchedArtwork, error)
	PrettyFileName(artwork shared.ArtworkLike, picture shared.PictureLike) string
}

// ArtworkTagSearcher 可选接口: 支持按 tag 搜索新作品的源实现它。
// 通过类型断言使用, 不影响其他源。
type ArtworkTagSearcher interface {
	SearchArtworksByTags(ctx context.Context, tags []string, limit int) ([]*dto.FetchedArtwork, error)
}

// ArtworkTagSearcherOrdered 可选接口: 支持按 tag 搜索且可指定排序的源。
// 实现它的源可获得更精细的推荐排序 (热门/最新)。
type ArtworkTagSearcherOrdered interface {
	SearchArtworksByTagsOrdered(ctx context.Context, tags []string, limit int, order string) ([]*dto.FetchedArtwork, error)
}

// ArtworkTagSearcherOrderedWithMode 可选接口: 在指定排序的基础上额外支持 R18 过滤模式。
// r18Mode: all(全部) / safe(全年龄) / r18(仅 R18)。
type ArtworkTagSearcherOrderedWithMode interface {
	SearchArtworksByTagsOrderedWithMode(ctx context.Context, tags []string, limit int, order, r18Mode string) ([]*dto.FetchedArtwork, error)
}

// ArtistArtworkLister 可选接口: 支持根据画师主页链接列出其全部作品链接。
type ArtistArtworkLister interface {
	// MatchArtistPageURL 若 text 包含画师主页链接, 返回规范化的主页链接。
	MatchArtistPageURL(text string) (string, bool)
	// FetchArtistArtworks 返回画师主页下全部作品的完整链接 (limit<=0 表示全部)。
	FetchArtistArtworks(ctx context.Context, artistPageURL string, limit int) ([]string, error)
}
