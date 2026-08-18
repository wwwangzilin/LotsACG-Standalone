package service

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

type stubArtworkSource struct{}

func (stubArtworkSource) GetArtworkInfo(ctx context.Context, sourceUrl string) (*dto.FetchedArtwork, error) {
	return nil, nil
}

func (stubArtworkSource) MatchesSourceURL(sourceUrl string) (string, bool) {
	return sourceUrl, strings.Contains(sourceUrl, "pixiv.net")
}

func (stubArtworkSource) FetchNewArtworks(ctx context.Context, limit int) ([]*dto.FetchedArtwork, error) {
	return nil, nil
}

func (stubArtworkSource) PrettyFileName(artwork shared.ArtworkLike, picture shared.PictureLike) string {
	return ""
}

func TestFindSourceURLsExtractsAllSupportedURLs(t *testing.T) {
	srv := &Service{sources: map[shared.SourceType]source.ArtworkSource{
		shared.SourceTypePixiv: stubArtworkSource{},
	}}

	got := srv.FindSourceURLs("先看 https://pixiv.net/artworks/1，然后再看 https://pixiv.net/artworks/2")
	want := []string{"https://pixiv.net/artworks/1", "https://pixiv.net/artworks/2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected urls: got %v want %v", got, want)
	}
}
