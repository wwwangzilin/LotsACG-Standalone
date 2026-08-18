package pixiv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"strings"

	"github.com/goccy/go-json"
	"github.com/imroc/req/v3"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/reutil"
)

func getPid(url string) string {
	matchUrl := sourceReg.FindString(url)
	id, ok := reutil.GetLatestNumberFromString(matchUrl)
	if !ok {
		return ""
	}
	return id
}

func doReqAjaxResp(ctx context.Context, sourceURL string, client *req.Client) (*PixivAjaxResp, error) {
	id := getPid(sourceURL)
	if id == "" {
		return nil, oops.New("invalid pixiv URL, cannot find artwork ID")
	}
	ajaxURL := "https://www.pixiv.net/ajax/illust/" + id
	resp, err := client.R().SetContext(ctx).Get(ajaxURL)
	if err != nil {
		return nil, err
	}
	body, err := respBodyBytes(resp)
	if err != nil {
		return nil, err
	}
	var pixivAjaxResp PixivAjaxResp
	err = json.Unmarshal(body, &pixivAjaxResp)
	if err != nil {
		return nil, ErrUnmarshalPixivAjax
	}
	return &pixivAjaxResp, nil
}

func reqAjaxResp(ctx context.Context, sourceURL string, client *req.Client) (*PixivAjaxResp, error) {
	resp, err := doReqAjaxResp(ctx, sourceURL, client)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func doReqIllustPages(ctx context.Context, sourceURL string, client *req.Client) (*PixivIllustPages, error) {
	ajaxURL := "https://www.pixiv.net/ajax/illust/" + getPid(sourceURL) + "/pages?lang=zh"
	resp, err := client.R().SetContext(ctx).Get(ajaxURL)
	if err != nil {
		return nil, err
	}
	body, err := respBodyBytes(resp)
	if err != nil {
		return nil, err
	}
	var pixivIllustPages PixivIllustPages
	err = json.Unmarshal(body, &pixivIllustPages)
	if err != nil {
		return nil, err
	}
	return &pixivIllustPages, nil
}

func reqIllustPages(ctx context.Context, sourceURL string, client *req.Client) (*PixivIllustPages, error) {
	resp, err := doReqIllustPages(ctx, sourceURL, client)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func doReqUgoiraMeta(ctx context.Context, sourceURL string, client *req.Client) (*PixivUgoiraMeta, error) {
	ajaxURL := "https://www.pixiv.net/ajax/illust/" + getPid(sourceURL) + "/ugoira_meta?lang=zh"
	resp, err := client.R().SetContext(ctx).Get(ajaxURL)
	if err != nil {
		return nil, err
	}
	body, err := respBodyBytes(resp)
	if err != nil {
		return nil, err
	}
	var pixivUgoiraMeta PixivUgoiraMeta
	err = json.Unmarshal(body, &pixivUgoiraMeta)
	if err != nil {
		return nil, err
	}
	return &pixivUgoiraMeta, nil
}

func reqUgoiraMeta(ctx context.Context, sourceURL string, client *req.Client) (*PixivUgoiraMeta, error) {
	resp, err := doReqUgoiraMeta(ctx, sourceURL, client)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *Pixiv) fetchNewArtworksForRSSURL(ctx context.Context, rssURL string, limit int) ([]*dto.FetchedArtwork, error) {
	var lastErr error
	for i := 0; i < len(p.reqClients); i++ {
		client := p.nextClient()
		resp, err := client.R().SetContext(ctx).Get(rssURL)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv rss request failed with account %d: %v", i+1, err)
			continue
		}
		bodyBytes, err := respBodyBytes(resp)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv rss decompress failed with account %d: %v", i+1, err)
			continue
		}
		body := string(bodyBytes)
		rsssum := sha256.Sum256(bodyBytes)
		fingerprint := hex.EncodeToString(rsssum[:])
		cacheKey := pixivRSSCacheKey(rssURL)

		if cacheEntry, err := kvstor.Get[pixivRSSCacheEntry](ctx, cacheKey); err == nil {
			if cacheEntry.Signature == fingerprint {
				if limit > 0 && len(cacheEntry.Artworks) > limit {
					return cacheEntry.Artworks[:limit], nil
				}
				return cacheEntry.Artworks, nil
			}
		}

		var pixivRss *PixivRss
		if err := xml.NewDecoder(strings.NewReader(body)).Decode(&pixivRss); err != nil {
			lastErr = err
			log.Warnf("pixiv rss decode failed with account %d: %v", i+1, err)
			continue
		}

		artworks := make([]*dto.FetchedArtwork, 0)
		for idx, item := range pixivRss.Channel.Items {
			if idx >= limit {
				break
			}
			ajaxResp, err := reqAjaxResp(ctx, item.Link, client)
			if err != nil {
				log.Warnf("pixiv rss item request failed with account %d for %s: %v", i+1, item.Link, err)
				continue
			}
			artwork, err := ajaxResp.ToArtwork(ctx, client, p.cfg.ImgProxy)
			if err != nil {
				log.Warnf("pixiv rss item conversion failed with account %d for %s: %v", i+1, item.Link, err)
				continue
			}
			artworks = append(artworks, artwork)
		}

		if len(artworks) > 0 || len(pixivRss.Channel.Items) == 0 {
			entry := pixivRSSCacheEntry{Signature: fingerprint, Artworks: artworks}
			if err := kvstor.Set(ctx, cacheKey, entry); err != nil {
				log.Warn("pixiv rss cache store failed", "url", rssURL, "err", err)
			}
		}
		return artworks, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, oops.New("no pixiv accounts available")
}

func pixivRSSCacheKey(rssURL string) string {
	sum := sha256.Sum256([]byte(rssURL))
	return "pixiv:rss:" + hex.EncodeToString(sum[:])
}

type pixivRSSCacheEntry struct {
	Signature string
	Artworks  []*dto.FetchedArtwork
}
