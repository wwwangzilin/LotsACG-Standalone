package misskey

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/dto"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
)

var (
	noteURLRegexp = regexp.MustCompile(`/notes/([a-zA-Z0-9]+)`)
)

// Misskey 实现 Misskey (联邦宇宙) 数据源: 通过公开 API 拉取带图片的帖子。
type Misskey struct {
	reqClient *req.Client
	cfg       runtimecfg.SourceMisskeyConfig
	instance  string
}

// misskeyNote Misskey note (帖子) 的 API 响应结构 (仅需要的字段)。
type misskeyNote struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
	User      struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user"`
	Files []struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		Thumbnail string `json:"thumbnailUrl"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"files"`
	Tags []string `json:"tags"`
}

func Init() {
	cfg := runtimecfg.Get().Source.Misskey
	if cfg.Disable || cfg.Instance == "" {
		return
	}
	client := req.C().ImpersonateChrome().SetCommonRetryCount(2)
	if runtimecfg.Get().Source.Proxy != "" {
		client.SetProxyURL(runtimecfg.Get().Source.Proxy)
	}
	source.Register(shared.SourceTypeMisskey, func() source.ArtworkSource {
		return &Misskey{
			cfg:       runtimecfg.Get().Source.Misskey,
			reqClient: client,
			instance:  strings.TrimSuffix(cfg.Instance, "/"),
		}
	})
}

// fetchTimeline 拉取指定用户的笔记时间线 (带图片的帖子)。
func (m *Misskey) fetchTimeline(ctx context.Context, userID string, limit int) ([]misskeyNote, error) {
	if limit <= 0 {
		limit = 20
	}
	body := map[string]any{
		"limit": limit,
	}
	endpoint := "/api/users/notes"
	if userID == "" {
		// 全局时间线 (需要认证, 大多数实例不可用); 用 features 或公共时间线兜底
		endpoint = "/api/notes/featured"
	} else {
		body["userId"] = userID
	}
	resp, err := m.reqClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBodyJsonMarshal(body).
		Post(m.instance + endpoint)
	if err != nil {
		return nil, err
	}
	if resp.IsErrorState() {
		return nil, fmt.Errorf("misskey api error: %s", resp.Status)
	}
	var notes []misskeyNote
	if err := json.Unmarshal(resp.Bytes(), &notes); err != nil {
		return nil, fmt.Errorf("misskey response unmarshal: %w", err)
	}
	return notes, nil
}

// FetchNewArtworks 拉取新作品: 支持配置多个 user_id (逗号分隔), 或全局 featured。
func (m *Misskey) FetchNewArtworks(ctx context.Context, limit int) ([]*dto.FetchedArtwork, error) {
	userIDs := []string{""}
	if ids := strings.TrimSpace(m.cfg.UserID); ids != "" {
		userIDs = strings.Split(ids, ",")
	}
	artworks := make([]*dto.FetchedArtwork, 0, len(userIDs)*10)
	for _, uid := range userIDs {
		uid = strings.TrimSpace(uid)
		notes, err := m.fetchTimeline(ctx, uid, limit)
		if err != nil {
			log.Warn("misskey fetch failed", "user", uid, "err", err)
			continue
		}
		for _, note := range notes {
			aw := m.noteToArtwork(note)
			if aw != nil {
				artworks = append(artworks, aw)
			}
		}
	}
	if len(artworks) == 0 {
		return nil, fmt.Errorf("no misskey artworks fetched")
	}
	return artworks, nil
}

// GetArtworkInfo 根据 note URL 获取作品信息。
func (m *Misskey) GetArtworkInfo(ctx context.Context, sourceURL string) (*dto.FetchedArtwork, error) {
	noteID := GetNoteID(sourceURL)
	if noteID == "" {
		return nil, ErrInvalidMisskeyURL
	}
	body := map[string]any{"noteId": noteID}
	resp, err := m.reqClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBodyJsonMarshal(body).
		Post(m.instance + "/api/notes/show")
	if err != nil {
		return nil, err
	}
	if resp.IsErrorState() {
		return nil, fmt.Errorf("misskey api error: %s", resp.Status)
	}
	var note misskeyNote
	if err := json.Unmarshal(resp.Bytes(), &note); err != nil {
		return nil, err
	}
	return m.noteToArtwork(note), nil
}

func (m *Misskey) MatchesSourceURL(text string) (string, bool) {
	noteID := GetNoteID(text)
	if noteID == "" {
		return "", false
	}
	return fmt.Sprintf("%s/notes/%s", m.instance, noteID), true
}

// PrettyFileName implements source.ArtworkSource.
func (m *Misskey) PrettyFileName(artwork shared.ArtworkLike, picture shared.PictureLike) string {
	idStr := GetNoteID(artwork.GetSourceURL())
	ext, _ := strutil.GetFileExtFromURL(picture.GetOriginal())
	if idStr == "" {
		return fmt.Sprintf("misskey_%s%s", strutil.MD5Hash(picture.GetOriginal()), ext)
	}
	return fmt.Sprintf("misskey_%s_%d%s", idStr, picture.GetIndex(), ext)
}

// noteToArtwork 将 Misskey note 转换为 FetchedArtwork; 无图片文件的帖子返回 nil。
func (m *Misskey) noteToArtwork(note misskeyNote) *dto.FetchedArtwork {
	if len(note.Files) == 0 {
		return nil
	}
	// 只取图片文件
	pictures := make([]*dto.FetchedPicture, 0, len(note.Files))
	for _, f := range note.Files {
		if !strings.HasPrefix(f.Type, "image/") {
			continue
		}
		pictures = append(pictures, &dto.FetchedPicture{
			Thumbnail: f.Thumbnail,
			Original:  f.URL,
			Index:     uint(len(pictures)),
			Width:     uint(f.Width),
			Height:    uint(f.Height),
		})
	}
	if len(pictures) == 0 {
		return nil
	}
	title := note.Text
	if title == "" {
		title = note.User.Name + " 的帖子"
	}
	if len(title) > 80 {
		title = title[:80]
	}
	return &dto.FetchedArtwork{
		Artist: &dto.FetchedArtist{
			Name:     note.User.Name,
			Type:     shared.SourceTypeMisskey,
			UID:      note.User.ID,
			Username: note.User.Username,
		},
		Title:       title,
		Description: note.Text,
		SourceType:  shared.SourceTypeMisskey,
		SourceURL:   fmt.Sprintf("%s/notes/%s", m.instance, note.ID),
		Tags:        note.Tags,
		Pictures:    pictures,
		CreateDate:  note.CreatedAt,
	}
}

// GetNoteID 从 URL 提取 note ID。
func GetNoteID(text string) string {
	if m := noteURLRegexp.FindStringSubmatch(text); len(m) > 1 {
		return m[1]
	}
	return ""
}

// ErrInvalidMisskeyURL 无效的 Misskey URL。
var ErrInvalidMisskeyURL = fmt.Errorf("invalid misskey note url")

var _ = url.URL{} // keep import
