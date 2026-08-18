package metautil

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

type MetaData struct {
	channelChatID    telego.ChatID
	r18ChannelChatID telego.ChatID
	groupChatID      telego.ChatID
	botUsername      string
	siteUrl          string
	botId            int64
	allowedUsers     []int64
	// should not set manually
	channelAvailable bool
}

func (m *MetaData) ChannelChatID() telego.ChatID {
	return m.channelChatID
}

func (m *MetaData) R18ChannelChatID() telego.ChatID {
	return m.r18ChannelChatID
}

func (m *MetaData) GroupChatID() telego.ChatID {
	return m.groupChatID
}

func (m *MetaData) BotUsername() string {
	return m.botUsername
}

func (m *MetaData) ChannelAvailable() bool {
	return m.channelAvailable
}

func (m *MetaData) SiteURL() string {
	return m.siteUrl
}

func (m *MetaData) BotID() int64 {
	return m.botId
}

// AllowedUsers 返回配置中登记的用户白名单 (admins + allowed_users)。
// 为空表示未启用白名单模式。
func (m *MetaData) AllowedUsers() []int64 {
	return m.allowedUsers
}

type MetaDataCtxKey struct{}

var contextKey = MetaDataCtxKey{}

type Option func(*MetaData)

func WithSiteURL(url string) Option {
	return func(m *MetaData) {
		m.siteUrl = strings.TrimRight(url, "/")
	}
}

func WithGroupChatID(id telego.ChatID) Option {
	return func(m *MetaData) {
		m.groupChatID = id
	}
}

func WithR18ChannelChatID(id telego.ChatID) Option {
	return func(m *MetaData) {
		m.r18ChannelChatID = id
	}
}

func WithBotID(id int64) Option {
	return func(m *MetaData) {
		m.botId = id
	}
}

// WithAllowedUsers 设置登记用户白名单 (admins + allowed_users)。
func WithAllowedUsers(ids []int64) Option {
	return func(m *MetaData) {
		m.allowedUsers = ids
	}
}

func NewMetaData(channelChatID telego.ChatID, botUsername string, opts ...Option) *MetaData {
	meta := &MetaData{
		channelChatID:    channelChatID,
		botUsername:      botUsername,
		channelAvailable: channelChatID.ID != 0 || channelChatID.Username != "",
	}
	for _, opt := range opts {
		opt(meta)
	}
	return meta
}

func FromContext(ctx context.Context) *MetaData {
	if meta, ok := ctx.Value(contextKey).(*MetaData); ok {
		return meta
	}
	return nil
}

func MustFromContext(ctx context.Context) *MetaData {
	meta := FromContext(ctx)
	if meta == nil {
		panic("metautil: MetaData not found in context")
	}
	return meta
}

func WithContext(ctx context.Context, meta *MetaData) context.Context {
	return context.WithValue(ctx, contextKey, meta)
}

func (m *MetaData) ResolvePostChatID(artwork shared.ArtworkLike) telego.ChatID {
	if artwork == nil {
		return m.channelChatID
	}
	if artwork.GetR18() {
		if m.r18ChannelChatID.ID != 0 || m.r18ChannelChatID.Username != "" {
			return m.r18ChannelChatID
		}
		return m.channelChatID
	}
	for _, tag := range artwork.GetTags() {
		if strings.EqualFold(tag, "R-18") || strings.EqualFold(tag, "R18") || strings.EqualFold(tag, "R-18G") || strings.EqualFold(tag, "R18G") {
			if m.r18ChannelChatID.ID != 0 || m.r18ChannelChatID.Username != "" {
				return m.r18ChannelChatID
			}
			break
		}
	}
	return m.channelChatID
}

func (m *MetaData) BotDeepLink(cmd string, params ...string) string {
	return fmt.Sprintf("https://t.me/%s/?start=%s_%s", m.botUsername, cmd, strings.Join(params, "_"))
}

func (m *MetaData) ChannelMessageURL(messageID int) string {
	if m.channelChatID.Username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", strings.TrimPrefix(m.channelChatID.Username, "@"), messageID)
	}
	return fmt.Sprintf("https://t.me/c/%s/%d", strings.TrimPrefix(fmt.Sprintf("%d", m.channelChatID.ID), "-100"), messageID)
}
