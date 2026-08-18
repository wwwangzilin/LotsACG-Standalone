package pixiv

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/imroc/req/v3"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// Pixiv App-API 客户端 (通过 refresh_token OAuth 访问收藏夹等)。
// 移植自 Pixiv-XP-Pusher 的 pixiv_client.py (AppPixivAPI)。

const (
	oauthTokenURL = "https://oauth.secure.pixiv.net/auth/token"
	appAPIBase    = "https://app-api.pixiv.net"
)

// 官方 client_id/client_secret (公开于 pixivpy/pixivpy_async)。
const (
	appClientID     = "MOBrBDS8blbauoSck0ZfDbtuzpyT"
	appClientSecret = "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj"
)

// AppAPIClient 是 Pixiv App-API 客户端。
type AppAPIClient struct {
	reqClient    *req.Client
	accessToken  string
	refreshToken string
	tokenExpiry  time.Time
	mu           sync.Mutex
}

// NewAppAPIClient 创建 App-API 客户端。
func NewAppAPIClient(refreshToken, proxy string) *AppAPIClient {
	c := req.C().
		SetLogger(log.Default()).
		SetCommonRetryCount(2).
		SetUserAgent("PixivIOSApp/7.13.3 (iOS 15.0; iPhone14,5)")
	if proxy != "" {
		c.SetProxyURL(proxy)
	}
	return &AppAPIClient{
		reqClient:    c,
		refreshToken: refreshToken,
	}
}

// oauthTokenResp 是 OAuth token 响应。
type oauthTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Response     struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	} `json:"response"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ensureToken 确保 access_token 有效, 过期或为空时用 refresh_token 刷新。
func (a *AppAPIClient) ensureToken(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		return nil
	}
	if a.refreshToken == "" {
		return oops.New("no pixiv refresh token configured")
	}

	form := url.Values{}
	form.Set("client_id", appClientID)
	form.Set("client_secret", appClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", a.refreshToken)

	var resp oauthTokenResp
	httpResp, err := a.reqClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBodyString(form.Encode()).
		Post(oauthTokenURL)
	if err != nil {
		return oops.Wrapf(err, "pixiv oauth token request failed")
	}
	body, err := respBodyBytes(httpResp)
	if err != nil {
		return oops.Wrapf(err, "pixiv oauth decompress failed")
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		previewLen := len(body)
		if previewLen > 200 {
			previewLen = 200
		}
		return oops.Wrapf(err, "pixiv oauth response unmarshal failed: %s", string(body[:previewLen]))
	}
	if httpResp.IsErrorState() {
		if resp.Error != nil {
			return oops.Errorf("pixiv oauth error: %s", resp.Error.Message)
		}
		return oops.Errorf("pixiv oauth http error: %d", httpResp.GetStatusCode())
	}
	if resp.AccessToken == "" {
		return oops.New("pixiv oauth returned empty access token")
	}
	a.accessToken = resp.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	if resp.RefreshToken != "" {
		a.refreshToken = resp.RefreshToken
	}
	return nil
}

// PixivBookmarkIllust 收藏夹中的单个作品 (用于画像构建)。
type PixivBookmarkIllust struct {
	ID          int64
	Title       string
	Tags        []string
	CreateDate  time.Time
	BookmarkCnt int
	UserID      int64
	UserName    string
}

// appBookmarkResp 收藏夹 API 响应。
type appBookmarkResp struct {
	Illusts []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Tags  []struct {
			Name string `json:"name"`
		} `json:"tags"`
		CreateDate     string `json:"create_date"`
		TotalBookmarks int    `json:"total_bookmarks"`
		User           struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	} `json:"illusts"`
	NextURL string `json:"next_url"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FetchBookmarks 获取用户公开收藏 (默认按时间倒序)。
// limit<=0 表示拉取全部 (最多 scanLimit 控制)。
func (a *AppAPIClient) FetchBookmarks(ctx context.Context, userID string, limit int) ([]PixivBookmarkIllust, error) {
	if err := a.ensureToken(ctx); err != nil {
		return nil, err
	}
	illusts := make([]PixivBookmarkIllust, 0)
	nextURL := ""

	for {
		if limit > 0 && len(illusts) >= limit {
			break
		}
		request := a.reqClient.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+a.accessToken)

		var resp appBookmarkResp
		var httpResp *req.Response
		var err error
		if nextURL != "" {
			httpResp, err = request.Get(nextURL)
		} else {
			httpResp, err = request.SetQueryParam("user_id", userID).
				SetQueryParam("restrict", "public").
				Get(appAPIBase + "/v1/user/bookmarks/illust")
		}
		if err != nil {
			return nil, oops.Wrapf(err, "pixiv bookmarks request failed")
		}
		body, err := respBodyBytes(httpResp)
		if err != nil {
			return nil, oops.Wrapf(err, "pixiv bookmarks decompress failed")
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, oops.Wrapf(err, "pixiv bookmarks unmarshal failed")
		}
		if httpResp.IsErrorState() {
			if resp.Error != nil {
				return nil, oops.Errorf("pixiv bookmarks error: %s", resp.Error.Message)
			}
			return nil, oops.Errorf("pixiv bookmarks http error: %d", httpResp.GetStatusCode())
		}
		if len(resp.Illusts) == 0 {
			break
		}
		for _, item := range resp.Illusts {
			if limit > 0 && len(illusts) >= limit {
				break
			}
			tags := make([]string, 0, len(item.Tags))
			for _, t := range item.Tags {
				if t.Name != "" {
					tags = append(tags, t.Name)
				}
			}
			createDate := time.Now()
			if item.CreateDate != "" {
				if t, err := time.Parse(time.RFC3339, item.CreateDate); err == nil {
					createDate = t
				}
			}
			illusts = append(illusts, PixivBookmarkIllust{
				ID:          item.ID,
				Title:       item.Title,
				Tags:        tags,
				CreateDate:  createDate,
				BookmarkCnt: item.TotalBookmarks,
				UserID:      item.User.ID,
				UserName:    item.User.Name,
			})
		}
		if resp.NextURL == "" {
			break
		}
		nextURL = resp.NextURL
	}
	return illusts, nil
}

// PixivUserID 返回配置的 Pixiv 用户 ID (字符串转 int64 校验)。
func PixivUserID(userID string) (int64, error) {
	if userID == "" {
		return 0, oops.New("pixiv user_id not configured")
	}
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, oops.Wrapf(err, "invalid pixiv user_id: %s", userID)
	}
	return id, nil
}

// appUserIllustResp 用户作品列表 API 响应。
type appUserIllustResp struct {
	Illusts []struct {
		ID int64 `json:"id"`
	} `json:"illusts"`
	NextURL string `json:"next_url"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FetchUserIllusts 获取指定用户的插画作品 ID 列表 (默认全部, 按时间倒序)。
// limit<=0 表示拉取全部。遇到 Pixiv 限流会自动退避重试。
func (a *AppAPIClient) FetchUserIllusts(ctx context.Context, userID string, limit int) ([]int64, error) {
	if err := a.ensureToken(ctx); err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	nextURL := ""

	for {
		if limit > 0 && len(ids) >= limit {
			break
		}
		pageIDs, newNext, err := a.fetchUserIllustsPage(ctx, userID, nextURL)
		if err != nil {
			return nil, err
		}
		if len(pageIDs) == 0 {
			break
		}
		for _, id := range pageIDs {
			if limit > 0 && len(ids) >= limit {
				break
			}
			ids = append(ids, id)
		}
		if newNext == "" {
			break
		}
		nextURL = newNext
	}
	return ids, nil
}

// fetchUserIllustsPage 拉取一页用户作品, 遇到限流(429 或空消息错误)时退避重试。
func (a *AppAPIClient) fetchUserIllustsPage(ctx context.Context, userID, nextURL string) ([]int64, string, error) {
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*3) * time.Second)
		}
		request := a.reqClient.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+a.accessToken)

		var resp appUserIllustResp
		var httpResp *req.Response
		var err error
		if nextURL != "" {
			httpResp, err = request.Get(nextURL)
		} else {
			httpResp, err = request.SetQueryParam("user_id", userID).
				SetQueryParam("type", "illust").
				Get(appAPIBase + "/v1/user/illusts")
		}
		if err != nil {
			return nil, "", oops.Wrapf(err, "pixiv user illusts request failed")
		}
		body, err := respBodyBytes(httpResp)
		if err != nil {
			return nil, "", oops.Wrapf(err, "pixiv user illusts decompress failed")
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, "", oops.Wrapf(err, "pixiv user illusts unmarshal failed")
		}
		if httpResp.IsErrorState() {
			msg := ""
			if resp.Error != nil {
				msg = resp.Error.Message
			}
			// 限流特征: HTTP 429 或带空消息的 error (Pixiv rate limit 返回空 message)
			isRateLimit := httpResp.GetStatusCode() == 429 || resp.Error != nil
			if isRateLimit && attempt < maxRetries {
				continue // 退避重试
			}
			// 附上响应体预览, 便于诊断 Pixiv 返回的具体内容
			preview := body
			if len(preview) > 300 {
				preview = preview[:300]
			}
			if resp.Error != nil {
				return nil, "", oops.Errorf("pixiv user illusts error: %s (status=%d body=%s)", msg, httpResp.GetStatusCode(), string(preview))
			}
			return nil, "", oops.Errorf("pixiv user illusts http error: %d (body=%s)", httpResp.GetStatusCode(), string(preview))
		}
		pageIDs := make([]int64, 0, len(resp.Illusts))
		for _, item := range resp.Illusts {
			if item.ID > 0 {
				pageIDs = append(pageIDs, item.ID)
			}
		}
		return pageIDs, resp.NextURL, nil
	}
	return nil, "", oops.New("pixiv user illusts: exhausted retries")
}
