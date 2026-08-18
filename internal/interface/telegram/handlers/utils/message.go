package utils

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

func ReplyMessageWithHTML(ctx *telegohandler.Context, message telego.Message, text string) (*telego.Message, error) {
	return ctx.Bot().SendMessage(ctx, telegoutil.Message(message.Chat.ChatID(), text).WithReplyParameters(
		&telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	).WithParseMode(telego.ModeHTML))
}

func ReplyMessage(ctx *telegohandler.Context, message telego.Message, text string) (*telego.Message, error) {
	return ctx.Bot().SendMessage(ctx, telegoutil.Message(message.Chat.ChatID(), text).WithReplyParameters(
		&telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	))
}

// EditMessage 编辑一条已发送的消息 (HTML)。
func EditMessage(ctx *telegohandler.Context, msg *telego.Message, text string) {
	if msg == nil {
		return
	}
	_, _ = ctx.Bot().EditMessageText(ctx, telegoutil.EditMessageText(msg.Chat.ChatID(), msg.MessageID, text).WithParseMode(telego.ModeHTML))
}

func FindSourceURLInMessage(serv *service.Service, message *telego.Message) string {
	urls := FindSourceURLsInMessage(serv, message)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func FindSourceURLsInMessage(serv *service.Service, message *telego.Message) []string {
	if message == nil {
		return nil
	}
	var sb strings.Builder
	sb.WriteString(message.Text)
	sb.WriteString(" ")
	sb.WriteString(message.Caption)
	for _, entity := range message.Entities {
		if entity.Type == telego.EntityTypeTextLink {
			sb.WriteString(entity.URL)
			sb.WriteString(" ")
		}
	}
	for _, entity := range message.CaptionEntities {
		if entity.Type == telego.EntityTypeTextLink {
			sb.WriteString(entity.URL)
			sb.WriteString(" ")
		}
	}
	return serv.FindSourceURLs(sb.String())
}

// FindArtistPageURLInMessage 从消息(文本/说明/链接实体)中查找画师主页链接。
func FindArtistPageURLInMessage(serv *service.Service, message *telego.Message) string {
	if message == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(message.Text)
	sb.WriteString(" ")
	sb.WriteString(message.Caption)
	for _, entity := range message.Entities {
		if entity.Type == telego.EntityTypeTextLink {
			sb.WriteString(entity.URL)
			sb.WriteString(" ")
		}
	}
	for _, entity := range message.CaptionEntities {
		if entity.Type == telego.EntityTypeTextLink {
			sb.WriteString(entity.URL)
			sb.WriteString(" ")
		}
	}
	return serv.FindArtistPageURL(sb.String())
}

var tagCharsReplacer = strings.NewReplacer(
	":", "_",
	"：", "_",
	"-", "_",
	"（", "_",
	"）", "_",
	"「", "_",
	"」", "_",
	"*", "_",
	"?", "",
	"/", " #",
	" ", "_",
)

var htmlEscapeReplacer = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
)

// https://core.telegram.org/bots/api#html-style
func EscapeHTML(s string) string {
	return htmlEscapeReplacer.Replace(s)
}

const defaultArtworkTemplate = `
<a href='{{.SourceURL}}'><b>{{.Title}}</b></a> / <b>{{.ArtistName}}</b>
{{- if .Description }}
<blockquote expandable=true>{{.Description}}</blockquote>
{{- end }}
{{- if .Tags }}
<blockquote expandable=true>{{.Tags}}</blockquote>
{{- end }}
`

type ArtworkCaptionData struct {
	ID          string
	SourceURL   string
	Title       string
	ArtistName  string
	Description template.HTML
	Tags        template.HTML
	// 用于对未发布到频道的作品的标记
	IsCache bool
}

func ArtworkHTMLCaption(artwork shared.ArtworkLike) string {
	tmplStr := defaultArtworkTemplate
	if runtimecfg.Get().Telegram.CaptionTemplate != "" {
		tmplStr = runtimecfg.Get().Telegram.CaptionTemplate
	}
	tmpl, err := template.New("artwork").Parse(tmplStr)
	if err != nil {
		log.Errorf("parse artwork caption template: %s", err)
		return artworkHTMLCaptionFallback(artwork)
	}

	sourceUrl := artwork.GetSourceURL()
	title := EscapeHTML(artwork.GetTitle())
	artistName := EscapeHTML(artwork.GetArtist().GetName())
	description := EscapeHTML(strutil.Ellipsis(artwork.GetDescription(), 500))

	tags := ""
	for _, tag := range artwork.GetTags() {
		if len(tags)+len(tag) > 200 {
			break
		}
		tag = tagCharsReplacer.Replace(tag)
		tag = strings.Trim(tag, "_")
		tags += "#" + strings.TrimSpace(EscapeHTML(tag)) + " "
	}

	cached, ok1 := artwork.(*entity.CachedArtwork)
	if ok1 {
		ok1 = cached.Status == shared.ArtworkStatusCached
	}
	_, ok2 := artwork.(*entity.CachedArtworkData)
	isCache := ok1 || ok2
	data := ArtworkCaptionData{
		SourceURL:  sourceUrl,
		Title:      title,
		ArtistName: artistName,
		IsCache:    isCache,
	}
	if description != "" {
		data.Description = template.HTML(description)
	}
	if tags != "" {
		data.Tags = template.HTML(tags)
	}
	data.ID = artwork.GetID()

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		log.Errorf("execute artwork caption template: %s", err)
		return artworkHTMLCaptionFallback(artwork)
	}

	return strings.TrimSpace(sb.String())
}

func artworkHTMLCaptionFallback(artwork shared.ArtworkLike) string {
	tmpl := "<a href='%s'><b>%s</b></a> / <b>%s</b>"
	sourceUrl := artwork.GetSourceURL()
	title := EscapeHTML(artwork.GetTitle())
	artistName := EscapeHTML(artwork.GetArtist().GetName())
	description := EscapeHTML(strutil.Ellipsis(artwork.GetDescription(), 500))

	tags := ""
	for _, tag := range artwork.GetTags() {
		if len(tags)+len(tag) > 200 {
			break
		}
		tag = tagCharsReplacer.Replace(tag)
		tag = strings.Trim(tag, "_")
		tags += "#" + strings.TrimSpace(EscapeHTML(tag)) + " "
	}

	args := []any{sourceUrl, title, artistName}

	if description != "" {
		tmpl += "\n<blockquote expandable=true>%s</blockquote>"
		args = append(args, description)
	}
	if tags != "" {
		tmpl += "\n<blockquote expandable=true>%s</blockquote>"
		args = append(args, tags)
	}

	caption := fmt.Sprintf(tmpl, args...)
	return caption
}

func GetMssageOriginChannel(message *telego.Message) *telego.MessageOriginChannel {
	if message.ForwardOrigin == nil {
		return nil
	}
	if message.ForwardOrigin.OriginType() == telego.OriginTypeChannel {
		return message.ForwardOrigin.(*telego.MessageOriginChannel)
	} else {
		return nil
	}
}

func GetPostedArtworkInlineKeyboardButton(artwork *entity.Artwork, meta *metautil.MetaData) []telego.InlineKeyboardButton {
	var detailsURL string
	if meta.SiteURL() != "" {
		detailsURL = fmt.Sprintf("%s/artwork/%s", meta.SiteURL(), artwork.ID.Hex())
	} else {
		detailsURL = artwork.SourceURL
	}
	hasValidTelegramInfo := meta.ChannelChatID().ID != 0 || meta.ChannelChatID().Username != ""
	if hasValidTelegramInfo && artwork.FirstMedia().GetTelegramInfo().MessageID(meta.ChannelChatID().ID) != 0 {
		detailsURL = meta.ChannelMessageURL(artwork.FirstMedia().GetTelegramInfo().MessageID(meta.ChannelChatID().ID))
	}
	return []telego.InlineKeyboardButton{
		telegoutil.InlineKeyboardButton("详情").WithURL(detailsURL),
		telegoutil.InlineKeyboardButton("原图").WithURL(meta.BotDeepLink("files", artwork.ID.Hex())),
	}
}

func GetMessagePhotoFile(ctx *telegohandler.Context, message *telego.Message) ([]byte, error) {
	if len(message.Photo) == 0 {
		return nil, oops.New("not a photo message")
	}
	size := message.Photo[len(message.Photo)-1]
	tfile, err := ctx.Bot().GetFile(ctx, &telego.GetFileParams{FileID: size.FileID})
	if err != nil {
		return nil, oops.Wrapf(err, "get file info failed")
	}
	dlUrl := ctx.Bot().FileDownloadURL(tfile.FilePath)
	// file, clean, err := httpclient.DownloadWithCache(ctx, dlUrl, nil)
	// if err != nil {
	// 	return nil, oops.Wrapf(err, "download file failed")
	// }
	// defer clean()
	// defer file.Close()
	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	return nil, oops.Wrapf(err, "read file failed")
	// }
	// return data, nil
	return telegoutil.DownloadFile(dlUrl)
}
