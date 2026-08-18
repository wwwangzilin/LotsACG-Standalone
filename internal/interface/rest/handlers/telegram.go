package handlers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
)

type RequestSendArtworkInfoByTelegramBot struct {
	SourceURL     string `json:"source_url" query:"source_url" form:"source_url" validate:"required"`
	ChatID        int64  `json:"chat_id" query:"chat_id" form:"chat_id" validate:"required"`
	AppendCaption string `json:"append_caption" query:"append_caption" form:"append_caption"`
}

// HandleSendArtworkInfoByTelegramBot 将指定来源链接的作品信息发送到指定群/用户。
// 使用已配置的 bot, 无需 API Key。
func HandleSendArtworkInfoByTelegramBot(ctx fiber.Ctx) error {
	bot, ok := common.GetState[common.TelegramBot](ctx, common.StateKeyTelegramBot)
	if !ok {
		return common.NewError(fiber.StatusNotFound, "telegram bot is not enabled")
	}
	req := new(RequestSendArtworkInfoByTelegramBot)
	if err := ctx.Bind().All(req); err != nil {
		return err
	}
	// current implement of SendArtworkInfo use a buffered channel, so it will return immediately and run in the background.
	// thus we should use context.Background() here.
	go bot.SendArtworkInfo(context.Background(), req.SourceURL, req.ChatID, req.AppendCaption)
	return ctx.JSON(common.NewSuccess("ok"))
}

// RequestPostArtworkToChannel 由外部 (如 XP-Pusher) 请求将作品发布到主频道。
type RequestPostArtworkToChannel struct {
	SourceURL string `json:"source_url" query:"source_url" form:"source_url" validate:"required"`
}

// HandlePostArtworkToChannel 将指定来源链接的作品发布到主频道。
// 供 XP-Pusher 的「推送到群」按钮调用。使用已配置的 bot, 无需 API Key。
func HandlePostArtworkToChannel(ctx fiber.Ctx) error {
	bot, ok := common.GetState[common.TelegramBot](ctx, common.StateKeyTelegramBot)
	if !ok {
		return common.NewError(fiber.StatusNotFound, "telegram bot is not enabled")
	}
	req := new(RequestPostArtworkToChannel)
	if err := ctx.Bind().All(req); err != nil {
		return err
	}
	if err := bot.PostArtworkToChannel(context.Background(), req.SourceURL); err != nil {
		return common.NewError(fiber.StatusInternalServerError, "post artwork to channel failed: "+err.Error())
	}
	return ctx.JSON(common.NewSuccess("ok"))
}

// HandleBotStatus 返回 Telegram bot 运行状态, 供设置页展示。
func HandleBotStatus(ctx fiber.Ctx) error {
	bot, ok := common.GetState[common.TelegramBot](ctx, common.StateKeyTelegramBot)
	if !ok {
		return common.NewError(fiber.StatusNotFound, "telegram bot is not enabled")
	}
	return ctx.JSON(common.NewSuccess(bot.Status(ctx.RequestCtx())))
}
