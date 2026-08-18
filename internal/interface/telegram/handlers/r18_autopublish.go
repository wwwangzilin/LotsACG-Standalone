package handlers

import (
	"github.com/mymmrac/telego/telegohandler"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// autoPublishToR18Channel 将已发布到主频道的作品自动发布到 R18 频道,
// 并记录 R18 频道的消息信息。返回错误表示未满足自动发布条件或发布失败。
func autoPublishToR18Channel(ctx *telegohandler.Context, serv *service.Service, artwork *entity.Artwork) error {
	meta := metautil.MustFromContext(ctx)
	r18Ch := meta.R18ChannelChatID()
	if r18Ch.ID == 0 && r18Ch.Username == "" {
		return oops.New("r18 channel not configured")
	}
	if artwork == nil || artwork.FirstMedia() == nil {
		return oops.New("artwork has no media")
	}
	// 已发布到主频道才自动发布到 R18 频道
	if artwork.FirstMedia().GetTelegramInfo().MessageID(meta.ChannelChatID().ID) == 0 {
		return oops.New("artwork not posted to main channel")
	}
	// 未发布到 R18 频道
	if artwork.FirstMedia().GetTelegramInfo().MessageID(r18Ch.ID) != 0 {
		return oops.New("artwork already posted to r18 channel")
	}

	results, err := utils.SendArtworkMediaGroup(ctx, ctx.Bot(), serv, meta, r18Ch, artwork)
	if err != nil {
		return oops.Wrapf(err, "send artwork to r18 channel failed")
	}
	// 记录 R18 频道消息信息
	for _, r := range results {
		msgID := r.Message.GetMessageID()
		groupID := r.Message.MediaGroupID
		switch r.Type {
		case utils.MediaResultTypePhoto:
			if len(artwork.Pictures) > r.Index {
				p := artwork.Pictures[r.Index]
				tg := p.GetTelegramInfo()
				tg.SetMessage(r18Ch.ID, msgID, groupID)
				tg.SetFileID(meta.BotID(), shared.TelegramMediaTypePhoto, r.FileID)
				if err := serv.UpdatePictureTelegramInfo(ctx, p.ID, &tg); err != nil {
					log.Warn("auto publish: update picture telegram info failed", "id", p.ID, "err", err)
				}
			}
		case utils.MediaResultTypeUgoira:
			if len(artwork.UgoiraMetas) > r.Index {
				u := artwork.UgoiraMetas[r.Index]
				tg := u.GetTelegramInfo()
				tg.SetMessage(r18Ch.ID, msgID, groupID)
				tg.SetFileID(meta.BotID(), shared.TelegramMediaTypeVideo, r.FileID)
				if err := serv.UpdateUgoiraTelegramInfo(ctx, u.ID, &tg); err != nil {
					log.Warn("auto publish: update ugoira telegram info failed", "id", u.ID, "err", err)
				}
			}
		case utils.MediaResultTypeVideo:
			if len(artwork.Videos) > r.Index {
				v := artwork.Videos[r.Index]
				tg := v.GetTelegramInfo()
				tg.SetMessage(r18Ch.ID, msgID, groupID)
				tg.SetFileID(meta.BotID(), shared.TelegramMediaTypeVideo, r.FileID)
				if err := serv.UpdateVideoTelegramInfo(ctx, v.ID, &tg); err != nil {
					log.Warn("auto publish: update video telegram info failed", "id", v.ID, "err", err)
				}
			}
		}
	}
	return nil
}
