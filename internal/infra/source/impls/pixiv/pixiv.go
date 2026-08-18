package pixiv

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/oops"
	config "github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"

	"github.com/imroc/req/v3"
)

type Pixiv struct {
	reqClients []*req.Client
	cfg        config.SourcePixivConfig
	clientIdx  int
}

func Init() {
	cfg := config.Get().Source.Pixiv
	if cfg.Disable {
		return
	}

	source.Register(shared.SourceTypePixiv, func() source.ArtworkSource {
		cfg := config.Get().Source.Pixiv
		clients := make([]*req.Client, 0, len(cfg.Accounts)+1)
		if len(cfg.Accounts) > 0 {
			for _, account := range cfg.Accounts {
				cookies := make([]*http.Cookie, 0, len(account.Cookies))
				for _, cookie := range account.Cookies {
					cookies = append(cookies, &http.Cookie{Name: cookie.Name, Value: cookie.Value})
				}
				c := req.C().ImpersonateChrome().SetCommonCookies(cookies...)
				c = c.SetLogger(log.Default()).EnableDebugLog().SetCommonRetryCount(3)
				if config.Get().Source.Proxy != "" {
					c.SetProxyURL(config.Get().Source.Proxy)
				}
				clients = append(clients, c)
			}
		} else {
			cookies := make([]*http.Cookie, 0, len(cfg.Cookies))
			for _, cookie := range cfg.Cookies {
				cookies = append(cookies, &http.Cookie{Name: cookie.Name, Value: cookie.Value})
			}
			c := req.C().ImpersonateChrome().SetCommonCookies(cookies...)
			c = c.SetLogger(log.Default()).EnableDebugLog().SetCommonRetryCount(3)
			if config.Get().Source.Proxy != "" {
				c.SetProxyURL(config.Get().Source.Proxy)
			}
			clients = append(clients, c)
		}
		return &Pixiv{cfg: cfg, reqClients: clients}
	})
}

func (p *Pixiv) FetchNewArtworks(ctx context.Context, limit int) ([]*dto.FetchedArtwork, error) {
	artworks := make([]*dto.FetchedArtwork, 0)
	errs := make([]error, 0)
	for _, url := range p.cfg.RssURLs {
		artworksForURL, err := p.fetchNewArtworksForRSSURL(ctx, url, limit)
		if err != nil {
			errs = append(errs, err)
		}
		artworks = append(artworks, artworksForURL...)
	}
	if len(errs) > 0 {
		return artworks, fmt.Errorf("fetching pixiv encountered %d errors: %v", len(errs), errs)
	}
	return artworks, nil
}

func (p *Pixiv) GetArtworkInfo(ctx context.Context, sourceURL string) (*dto.FetchedArtwork, error) {
	var lastErr error
	for i := 0; i < len(p.reqClients); i++ {
		client := p.nextClient()
		ajaxResp, err := reqAjaxResp(ctx, sourceURL, client)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv artwork info request failed with account %d: %v", i+1, err)
			continue
		}
		if ajaxResp.Err {
			lastErr = oops.Errorf("pixiv ajax response error: %s", ajaxResp.Message)
			log.Warnf("pixiv artwork info request returned error with account %d: %s", i+1, ajaxResp.Message)
			continue
		}
		artwork, err := ajaxResp.ToArtwork(ctx, client, p.cfg.ImgProxy)
		if err != nil {
			lastErr = err
			log.Warnf("pixiv artwork conversion failed with account %d: %v", i+1, err)
			continue
		}
		return artwork, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, oops.New("no pixiv accounts available")
}

func (p *Pixiv) MatchesSourceURL(text string) (string, bool) {
	pid := getPid(text)
	if pid == "" {
		return "", false
	}
	return "https://www.pixiv.net/artworks/" + pid, true
}

func (p *Pixiv) nextClient() *req.Client {
	if len(p.reqClients) == 0 {
		return nil
	}
	if len(p.reqClients) == 1 {
		return p.reqClients[0]
	}
	client := p.reqClients[p.clientIdx]
	p.clientIdx = (p.clientIdx + 1) % len(p.reqClients)
	return client
}

func (p *Pixiv) resetClientIndex() {
	if len(p.reqClients) == 0 {
		p.clientIdx = 0
		return
	}
	p.clientIdx = 0
}

func (p *Pixiv) PrettyFileName(artwork shared.ArtworkLike, picture shared.PictureLike) string {
	pid := getPid(artwork.GetSourceURL())
	ext, _ := strutil.GetFileExtFromURL(picture.GetOriginal())
	if pid != "" {
		return fmt.Sprintf("pixiv_%s_%d%s", pid, picture.GetIndex(), ext)
	}
	return fmt.Sprintf("pixiv_%s%s", strutil.MD5Hash(picture.GetOriginal()), ext)
}

// artistPageURLRegex 匹配 pixiv 画师主页链接, 如:
// https://www.pixiv.net/users/123456
// https://www.pixiv.net/en/users/123456/artworks
var artistPageURLRegex = regexp.MustCompile(`pixiv\.net/(?:[a-z]{2}/)?users/(\d+)`)

// MatchArtistPageURL 若 text 包含 pixiv 画师主页链接, 返回规范化的主页链接。
func (p *Pixiv) MatchArtistPageURL(text string) (string, bool) {
	m := artistPageURLRegex.FindStringSubmatch(text)
	if len(m) < 2 || m[1] == "" {
		return "", false
	}
	return "https://www.pixiv.net/users/" + m[1], true
}

// FetchArtistArtworks 返回指定画师主页下的全部作品完整链接。
// limit<=0 表示拉取全部。优先使用网页版接口 (cookie, 不走 OAuth, 避免 app-api 限流),
// 失败时回退到 app-api (需要 refresh_token)。结果缓存 30 分钟。
func (p *Pixiv) FetchArtistArtworks(ctx context.Context, artistPageURL string, limit int) ([]string, error) {
	// 从主页链接提取纯 user ID (注意: MatchArtistPageURL 返回的是规范化链接, 不能直接当 ID 用)
	m := artistPageURLRegex.FindStringSubmatch(artistPageURL)
	if len(m) < 2 || m[1] == "" {
		return nil, oops.New("not a pixiv artist page url")
	}
	userID := m[1]

	// 缓存: 同一画师 30 分钟内直接返回, 避免频繁请求触发限流
	cacheKey := "pixiv:artist:" + userID
	if cached, err := kvstor.Get[[]string](ctx, cacheKey); err == nil && len(cached) > 0 {
		return cached, nil
	}

	ids, err := p.fetchArtistIllustsViaWeb(ctx, userID)
	if err != nil || len(ids) == 0 {
		// 回退到 app-api (需要 refresh_token)
		cfg := config.Get().XPAIAPI.Pixiv
		if cfg.RefreshToken != "" {
			client := NewAppAPIClient(cfg.RefreshToken, config.Get().Source.Proxy)
			if apiIDs, apiErr := client.FetchUserIllusts(ctx, userID, limit); apiErr == nil && len(apiIDs) > 0 {
				ids = apiIDs
				err = nil
			}
		}
	}
	if err != nil {
		return nil, oops.Wrapf(err, "failed to fetch artist artworks")
	}
	urls := make([]string, 0, len(ids))
	for _, id := range ids {
		urls = append(urls, fmt.Sprintf("https://www.pixiv.net/artworks/%d", id))
	}
	if len(urls) > 0 {
		_ = kvstor.SetWithTTL(ctx, cacheKey, urls, 30*time.Minute)
	}
	return urls, nil
}

// artistProfileAllResp 是 pixiv 网页版 profile/all 接口的响应。
type artistProfileAllResp struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		Illusts map[string]any `json:"illusts"`
		Manga   map[string]any `json:"manga"`
	} `json:"body"`
}

// fetchArtistIllustsViaWeb 使用网页版 ajax 接口获取用户全部作品 ID (插画 + 漫画)。
// 使用现有 cookie 客户端, 不依赖 OAuth, 避免 app-api 限流。
func (p *Pixiv) fetchArtistIllustsViaWeb(ctx context.Context, userID string) ([]int64, error) {
	var lastErr error
	for i := 0; i < len(p.reqClients); i++ {
		client := p.nextClient()
		resp, err := client.R().
			SetContext(ctx).
			SetHeader("Referer", "https://www.pixiv.net/").
			SetHeader("Accept", "application/json").
			Get("https://www.pixiv.net/ajax/user/" + userID + "/profile/all?lang=zh")
		if err != nil {
			lastErr = oops.Wrapf(err, "pixiv profile/all request failed")
			continue
		}
		body, err := respBodyBytes(resp)
		if err != nil {
			lastErr = oops.Wrapf(err, "pixiv profile/all decompress failed")
			continue
		}
		var pr artistProfileAllResp
		if err := json.Unmarshal(body, &pr); err != nil {
			lastErr = oops.Wrapf(err, "pixiv profile/all unmarshal failed")
			continue
		}
		if pr.Error {
			lastErr = oops.Errorf("pixiv profile/all error: %s", pr.Message)
			continue
		}
		idSet := make(map[int64]struct{}, len(pr.Body.Illusts)+len(pr.Body.Manga))
		for idStr := range pr.Body.Illusts {
			if id, e := strconv.ParseInt(idStr, 10, 64); e == nil {
				idSet[id] = struct{}{}
			}
		}
		for idStr := range pr.Body.Manga {
			if id, e := strconv.ParseInt(idStr, 10, 64); e == nil {
				idSet[id] = struct{}{}
			}
		}
		ids := make([]int64, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		// 按 ID 降序 (最新作品在前)
		sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
		return ids, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, oops.New("no pixiv accounts available")
}
