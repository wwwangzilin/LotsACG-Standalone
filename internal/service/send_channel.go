package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
)

// SendChannel R18 规则取值。
const (
	// SendChannelR18Follow 跟随主频道分流 (默认): 不额外限制 R18。
	SendChannelR18Follow = "follow"
	// SendChannelR18Allow 允许 R18: R18 作品也会发到该频道。
	SendChannelR18Allow = "allow"
	// SendChannelR18Deny 拒绝 R18: R18 作品不发到该频道。
	SendChannelR18Deny = "deny"
	// SendChannelR18Only 仅 R18: 只发 R18 作品。
	SendChannelR18Only = "only"
)

// SendChannel 是一个自定义发送频道(群/频道), 支持按规则自动发布作品。
// 存储于 KV: 数据 key `sendchannel:<chatid>`, 列表索引 key `sendchannel:list`。
// 除主频道外, 作品发布时会按这里配置的规则自动推送到匹配的频道 (不落库)。
type SendChannel struct {
	ChatID         int64    `json:"chat_id"`
	Title          string   `json:"title"`
	Enabled        bool     `json:"enabled"`
	R18Mode        string   `json:"r18_mode"` // follow|allow|deny|only
	LinkOnly       bool     `json:"link_only"`
	IncludeTags    []string `json:"include_tags"`    // 标签白名单 (任一命中即通过, 空=不限制)
	ExcludeTags    []string `json:"exclude_tags"`    // 标签黑名单 (任一命中即拒绝)
	IncludeArtists []string `json:"include_artists"` // 画师白名单 (空=不限制)
	ExcludeArtists []string `json:"exclude_artists"` // 画师黑名单
	CreatedAt      int64    `json:"created_at"`
}

func sendChannelListKey() string {
	return "sendchannel:list"
}

func sendChannelKey(chatID int64) string {
	return fmt.Sprintf("sendchannel:%d", chatID)
}

// ListSendChannels 返回全部发送频道 (按添加顺序)。
func (s *Service) ListSendChannels(ctx context.Context) ([]*SendChannel, error) {
	ids, err := kvstor.Get[[]int64](ctx, sendChannelListKey())
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	chs := make([]*SendChannel, 0, len(ids))
	for _, id := range ids {
		ch, err := kvstor.Get[SendChannel](ctx, sendChannelKey(id))
		if err != nil {
			if errors.Is(err, errs.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		cc := ch
		chs = append(chs, &cc)
	}
	return chs, nil
}

// GetSendChannel 获取单个发送频道。
func (s *Service) GetSendChannel(ctx context.Context, chatID int64) (*SendChannel, error) {
	ch, err := kvstor.Get[SendChannel](ctx, sendChannelKey(chatID))
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// AddSendChannel 新增一个发送频道, 已存在则报错。
func (s *Service) AddSendChannel(ctx context.Context, ch *SendChannel) error {
	if ch == nil || ch.ChatID == 0 {
		return oops.New("chat id 不能为空")
	}
	if _, err := kvstor.Get[SendChannel](ctx, sendChannelKey(ch.ChatID)); err == nil {
		return oops.Errorf("频道 %d 已存在", ch.ChatID)
	} else if !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}
	if ch.Title == "" {
		ch.Title = fmt.Sprintf("频道 %d", ch.ChatID)
	}
	if ch.R18Mode == "" {
		ch.R18Mode = SendChannelR18Follow
	}
	if ch.CreatedAt == 0 {
		ch.CreatedAt = time.Now().Unix()
	}
	if err := kvstor.Set(ctx, sendChannelKey(ch.ChatID), *ch); err != nil {
		return err
	}
	ids, err := kvstor.Get[[]int64](ctx, sendChannelListKey())
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}
	if !slices.Contains(ids, ch.ChatID) {
		ids = append(ids, ch.ChatID)
		return kvstor.Set(ctx, sendChannelListKey(), ids)
	}
	return nil
}

// SaveSendChannel 更新发送频道 (不存在则新增)。
func (s *Service) SaveSendChannel(ctx context.Context, ch *SendChannel) error {
	if ch == nil || ch.ChatID == 0 {
		return oops.New("chat id 不能为空")
	}
	existing, err := kvstor.Get[SendChannel](ctx, sendChannelKey(ch.ChatID))
	if err == nil {
		ch.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}
	if ch.R18Mode == "" {
		ch.R18Mode = SendChannelR18Follow
	}
	if ch.Title == "" {
		ch.Title = fmt.Sprintf("频道 %d", ch.ChatID)
	}
	if ch.CreatedAt == 0 {
		ch.CreatedAt = time.Now().Unix()
	}
	if err := kvstor.Set(ctx, sendChannelKey(ch.ChatID), *ch); err != nil {
		return err
	}
	ids, err := kvstor.Get[[]int64](ctx, sendChannelListKey())
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}
	if !slices.Contains(ids, ch.ChatID) {
		ids = append(ids, ch.ChatID)
		return kvstor.Set(ctx, sendChannelListKey(), ids)
	}
	return nil
}

// RemoveSendChannel 删除一个发送频道。
func (s *Service) RemoveSendChannel(ctx context.Context, chatID int64) error {
	if err := kvstor.Delete(ctx, sendChannelKey(chatID)); err != nil {
		return err
	}
	ids, err := kvstor.Get[[]int64](ctx, sendChannelListKey())
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}
	ids = slices.DeleteFunc(ids, func(id int64) bool { return id == chatID })
	return kvstor.Set(ctx, sendChannelListKey(), ids)
}

// MatchSendChannels 返回启用且规则匹配的发送频道。
func (s *Service) MatchSendChannels(ctx context.Context, artwork shared.ArtworkLike) ([]*SendChannel, error) {
	chs, err := s.ListSendChannels(ctx)
	if err != nil {
		return nil, err
	}
	matched := make([]*SendChannel, 0, len(chs))
	for _, ch := range chs {
		if MatchSendChannelRule(ch, artwork) {
			matched = append(matched, ch)
		}
	}
	return matched, nil
}

// MatchSendChannelRule 判断一个作品是否满足发送频道的发布规则。
// 规则: 启用 → R18 模式 → 标签白/黑名单 → 画师白/黑名单。
func MatchSendChannelRule(ch *SendChannel, artwork shared.ArtworkLike) bool {
	if ch == nil || artwork == nil || !ch.Enabled {
		return false
	}
	r18 := artwork.GetR18()
	switch ch.R18Mode {
	case SendChannelR18Deny:
		if r18 {
			return false
		}
	case SendChannelR18Only:
		if !r18 {
			return false
		}
	}
	tags := artwork.GetTags()
	if len(ch.IncludeTags) > 0 {
		hit := false
		for _, want := range ch.IncludeTags {
			if slices.ContainsFunc(tags, func(t string) bool { return strings.EqualFold(t, want) }) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	for _, banned := range ch.ExcludeTags {
		if slices.ContainsFunc(tags, func(t string) bool { return strings.EqualFold(t, banned) }) {
			return false
		}
	}
	artistName := ""
	if artist := artwork.GetArtist(); artist != nil {
		artistName = artist.GetName()
	}
	if len(ch.IncludeArtists) > 0 {
		hit := false
		for _, want := range ch.IncludeArtists {
			if strings.EqualFold(strings.TrimSpace(artistName), strings.TrimSpace(want)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	for _, banned := range ch.ExcludeArtists {
		if strings.EqualFold(strings.TrimSpace(artistName), strings.TrimSpace(banned)) {
			return false
		}
	}
	return true
}
