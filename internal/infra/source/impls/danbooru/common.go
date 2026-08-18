package danbooru

import (
	"errors"
	"regexp"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/reutil"
)

var (
	danbooruSourceURLRegexp = regexp.MustCompile(`danbooru\.donmai\.us/(posts|post\/show)/\d+`)
	fakeArtist              = &dto.FetchedArtist{
		Name:     "Danbooru",
		Username: "Danbooru",
		UID:      "1",
		Type:     shared.SourceTypeDanbooru,
	}
	ErrInvalidDanbooruPostURL = errors.New("invalid danbooru post url")
	ErrDanbooruNoImage        = errors.New("danbooru post has no image")
)

func GetPostID(url string) string {
	matchUrl := danbooruSourceURLRegexp.FindString(url)
	if matchUrl == "" {
		return ""
	}
	id, ok := reutil.GetLatestNumberFromString(matchUrl)
	if !ok {
		return ""
	}
	return id
}
