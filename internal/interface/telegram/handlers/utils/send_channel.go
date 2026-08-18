package utils

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

// SendArtworkLinkOnly 只发送作品信息文本 + 原图链接 (不下载/上传图片)。
// 用于配置了 link_only 规则的发送频道, 节约流量。
func SendArtworkLinkOnly(ctx context.Context, bot *telego.Bot, serv *service.Service, meta *metautil.MetaData, chatID telego.ChatID, artwork shared.ArtworkLike) error {
	var b strings.Builder
	if title := strings.TrimSpace(artwork.GetTitle()); title != "" {
		b.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(title)))
	}
	if artist := artwork.GetArtist(); artist != nil {
		name := strings.TrimSpace(artist.GetName())
		if name == "" {
			name = strings.TrimSpace(artist.GetUserName())
		}
		if name != "" {
			b.WriteString(fmt.Sprintf("👤 %s\n", html.EscapeString(name)))
		}
	}
	if url := strings.TrimSpace(artwork.GetSourceURL()); url != "" {
		b.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">原图链接</a>\n", html.EscapeString(url)))
	}
	tags := artwork.GetTags()
	if len(tags) > 0 {
		max := len(tags)
		if max > 10 {
			max = 10
		}
		hs := make([]string, 0, max)
		for _, t := range tags[:max] {
			tag := strings.TrimSpace(t)
			if tag == "" {
				continue
			}
			tag = strings.ReplaceAll(tag, " ", "_")
			hs = append(hs, "#"+tag)
		}
		if len(hs) > 0 {
			b.WriteString(strings.Join(hs, " ") + "\n")
		}
	}
	if artwork.GetR18() {
		b.WriteString("🔞 R18\n")
	}
	if b.Len() == 0 {
		b.WriteString("作品链接\n")
	}
	_, err := bot.SendMessage(ctx, telegoutil.Message(chatID, strings.TrimRight(b.String(), "\n")).WithParseMode(telego.ModeHTML))
	return err
}
