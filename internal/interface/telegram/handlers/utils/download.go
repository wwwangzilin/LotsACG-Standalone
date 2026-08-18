package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/pixiv"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
)

// DownloadPixivImageWithFallback downloads a Pixiv image trying the configured
// proxy hosts in priority order (manyacg -> pixiv.cat -> i.muxmus.com -> official).
// Returns the downloaded file and the URL that succeeded.
func DownloadPixivImageWithFallback(ctx context.Context, imageURL string, attemptsPerSource int) (*osutil.File, string, error) {
	if attemptsPerSource <= 0 {
		attemptsPerSource = 3
	}
	proxyHosts := runtimecfg.Get().Source.Pixiv.ImgProxyHosts()
	candidates := pixiv.BuildPixivImageCandidates(imageURL, proxyHosts)

	var lastErr error
	for ci, candidate := range candidates {
		for retry := 0; retry < attemptsPerSource; retry++ {
			if retry > 0 {
				time.Sleep(time.Duration(retry) * time.Second)
			}
			f, err := httpclient.DownloadWithCache(ctx, candidate, nil)
			if err == nil {
				return f, candidate, nil
			}
			lastErr = err
			log.Warnf("download image via %s attempt %d failed: %v", candidate, retry+1, err)
		}
		if ci < len(candidates)-1 {
			log.Warnf("image download failed via %s, trying next source", candidate)
		}
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no image sources available for %s", imageURL)
}
