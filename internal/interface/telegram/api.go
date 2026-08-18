package telegram

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// Status 返回 bot 运行状态 (用户名、频道/群信息), 供 REST 设置页展示。
func (b *BotApp) Status(ctx context.Context) common.TelegramBotStatus {
	channel := b.meta.ChannelChatID()
	return common.TelegramBotStatus{
		BotUsername:  b.meta.BotUsername(),
		ChannelID:    channel.ID,
		ChannelName:  channel.Username,
		GroupID:      b.meta.GroupChatID().ID,
		R18ChannelID: b.meta.R18ChannelChatID().ID,
		Running:      true,
		AllowedUsers: b.meta.AllowedUsers(),
	}
}

func (b *BotApp) PostAndCreateArtwork(ctx context.Context, artwork *entity.CachedArtworkData) error {
	var adminId telego.ChatID
	if len(b.cfg.Admins) > 0 {
		adminId = telegoutil.ID(b.cfg.Admins[0])
	} else {
		adminIds, _ := b.serv.GetAdminUserIDs(ctx)
		if len(adminIds) > 0 {
			adminId = telegoutil.ID(adminIds[0])
		}
	}
	if err := utils.PostAndCreateArtwork(ctx, b.Bot(), b.serv, b.meta, artwork, adminId, b.meta.ChannelChatID(), 0); err != nil {
		return oops.Wrapf(err, "posting and creating artwork %s", artwork.SourceURL)
	}
	// 多频道: 按规则发布到自定义发送频道 (不落库)
	if err := b.publishToCustomChannels(ctx, artwork); err != nil {
		log.Warn("failed to publish to custom send channels", "err", err)
	}
	return nil
}

type artworkInfoTask struct {
	ctx           context.Context `msgpack:"-"`
	SourceURL     string          `msgpack:"source_url"`
	ChatID        int64           `msgpack:"chat_id"`
	AppendCaption string          `msgpack:"append_caption"`
}

// matches 判断两个任务是否表示同一请求 (用于从持久化队列精确移除)。
func (t artworkInfoTask) matches(o artworkInfoTask) bool {
	return t.SourceURL == o.SourceURL && t.ChatID == o.ChatID && t.AppendCaption == o.AppendCaption
}

func (b *BotApp) SendArtworkInfo(ctx context.Context, sourceUrl string, chatID int64, appendCaption string) {
	task := artworkInfoTask{
		ctx:           ctx,
		SourceURL:     sourceUrl,
		ChatID:        chatID,
		AppendCaption: appendCaption,
	}
	// 先持久化到 KV (意外退出后重启可恢复), 再入内存队列
	b.persistArtworkInfoTask(context.Background(), task)
	b.artworkInfoQueue <- task
}

// PostArtworkToChannel 将指定来源链接的作品发布到主频道 (供 XP-Pusher 等外部调用)。
func (b *BotApp) PostArtworkToChannel(ctx context.Context, sourceURL string) error {
	cachedArtwork, err := b.serv.GetOrFetchCachedArtwork(ctx, sourceURL)
	if err != nil {
		return oops.Wrapf(err, "failed to get or fetch cached artwork")
	}
	if cachedArtwork.Status != shared.ArtworkStatusCached {
		return oops.New("artwork already posted or being posted")
	}
	artwork := cachedArtwork.Artwork.Data()
	if artwork == nil || len(artwork.Pictures) == 0 {
		return oops.New("artwork has no pictures")
	}
	if err := utils.PostAndCreateArtwork(ctx, b.Bot(), b.serv, b.meta, artwork, telego.ChatID{}, b.meta.ChannelChatID(), 0); err != nil {
		return oops.Wrapf(err, "failed to post artwork to channel")
	}
	// 多频道: 按规则发布到自定义发送频道 (不落库)
	if err := b.publishToCustomChannels(ctx, artwork); err != nil {
		log.Warn("failed to publish to custom send channels", "err", err)
	}
	return nil
}

// publishToCustomChannels 把作品按各发送频道的规则发布到匹配的自定义频道 (不落库)。
// link_only 频道只发链接文本; 其他频道发送图片组。
func (b *BotApp) publishToCustomChannels(ctx context.Context, artwork *entity.CachedArtworkData) error {
	chs, err := b.serv.MatchSendChannels(ctx, artwork)
	if err != nil {
		return oops.Wrapf(err, "matching send channels")
	}
	for _, ch := range chs {
		chatID := telegoutil.ID(ch.ChatID)
		if ch.LinkOnly {
			if err := utils.SendArtworkLinkOnly(ctx, b.Bot(), b.serv, b.meta, chatID, artwork); err != nil {
				log.Warn("failed to send link-only to send channel", "chat_id", ch.ChatID, "title", ch.Title, "err", err)
				continue
			}
			log.Info("sent link-only to send channel", "chat_id", ch.ChatID, "title", ch.Title)
			continue
		}
		if _, err := utils.SendArtworkMediaGroup(ctx, b.Bot(), b.serv, b.meta, chatID, artwork); err != nil {
			log.Warn("failed to send media group to send channel", "chat_id", ch.ChatID, "title", ch.Title, "err", err)
			continue
		}
		log.Info("sent artwork to send channel", "chat_id", ch.ChatID, "title", ch.Title)
	}
	return nil
}

// SendArtworkNotification 实现 scheduler.ArtworkNotifier: 向用户推送新作品 (画师关注/标签订阅)。
func (b *BotApp) SendArtworkNotification(ctx context.Context, userID int64, sourceURL string) error {
	b.SendArtworkInfo(ctx, sourceURL, userID, "")
	return nil
}

// SendTextToUser 向指定用户发送一条文本消息。
func (b *BotApp) SendTextToUser(ctx context.Context, userID int64, text string) error {
	_, err := b.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telegoutil.ID(userID),
		Text:      text,
		ParseMode: telego.ModeHTML,
	})
	return err
}
