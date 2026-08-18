package handlers

import (
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// RedescribeArtwork 处理 /redescribe 指令:
// 使用 AI 根据作品标签生成描述, 并以引用文本格式追加到频道帖子内容下方。
func RedescribeArtwork(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionEditArtwork) {
		utils.ReplyMessage(ctx, message, "你没有编辑作品的权限")
		return nil
	}
	var sourceURL string
	if message.ReplyToMessage != nil {
		sourceURL = utils.FindSourceURLInMessage(serv, message.ReplyToMessage)
	} else {
		sourceURL = serv.FindSourceURL(message.Text)
	}
	if sourceURL == "" {
		helpText := `
[管理员] <b>使用 /redescribe 命令回复一条包含作品链接的消息, 或在参数中提供作品链接, 将使用 AI 生成描述并追加到频道帖子下方</b>

命令语法: /redescribe [作品链接]
`
		utils.ReplyMessageWithHTML(ctx, message, helpText)
		return nil
	}
	artwork, err := serv.GetArtworkByURL(ctx, sourceURL)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取作品信息失败: "+err.Error())
		return nil
	}
	meta, err := requireMeta(ctx)
	if err != nil {
		return err
	}
	msgID := artwork.FirstMedia().GetTelegramInfo().MessageID(meta.ChannelChatID().ID)
	if msgID == 0 {
		utils.ReplyMessage(ctx, message, "该作品未在频道发布")
		return nil
	}
	msg, err := utils.ReplyMessage(ctx, message, "正在生成描述...")
	if err != nil {
		return err
	}
	desc, err := serv.GenerateArtworkDescription(ctx, artwork)
	if err != nil {
		log.Errorf("failed to generate artwork description: %s", err)
		ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    msg.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "生成描述失败: " + err.Error(),
		})
		return nil
	}
	// 在原帖子内容下方追加引用文本 (blockquote)
	caption := utils.ArtworkHTMLCaption(artwork) + "\n\n<blockquote>" + utils.EscapeHTML(desc) + "</blockquote>"
	if len(caption) > 1000 {
		caption = caption[:1000]
	}
	if _, err := ctx.Bot().EditMessageCaption(ctx, &telego.EditMessageCaptionParams{
		ChatID:    meta.ChannelChatID(),
		MessageID: msgID,
		Caption:   caption,
		ParseMode: telego.ModeHTML,
	}); err != nil {
		ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    msg.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "更新频道帖子失败: " + err.Error(),
		})
		return nil
	}
	ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    msg.Chat.ChatID(),
		MessageID: msg.MessageID,
		Text:      fmt.Sprintf("已在频道帖子下方追加 AI 描述\n%s", desc),
	})
	return nil
}
