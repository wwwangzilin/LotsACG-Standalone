package handlers

import (
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/query"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
)

// GroupSearch 处理 /groupsearch 指令: 在数据库(群/频道已收录作品)中按关键词搜索。
// 支持 '|' 分隔或关系, 空格分隔与关系 (与 /random 相同)。
func GroupSearch(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	if len(args) == 0 {
		helpText := `
<b>使用 /groupsearch 命令在群组中搜索作品</b>

命令语法: /groupsearch 关键词

支持使用 '|' 分隔'或'关系, 使用 '空格' 分隔'与'关系, 示例:
/groupsearch 萝莉|白丝 猫耳|原创
`
		utils.ReplyMessageWithHTML(ctx, message, helpText)
		return nil
	}
	keyword := strings.Join(args, " ")
	textArray := strutil.ParseTo2DArray(keyword, "|", " ")

	artworks, err := serv.QueryArtworks(ctx, query.ArtworksDB{
		ArtworksFilter: query.ArtworksFilter{
			Keywords:   textArray,
			HasPicture: true,
		},
		Paginate: query.Paginate{
			Offset: 0,
			Limit:  10,
		},
	})
	if err != nil {
		utils.ReplyMessage(ctx, message, "搜索失败: "+err.Error())
		return oops.Wrapf(err, "group search failed")
	}
	if len(artworks) == 0 {
		utils.ReplyMessage(ctx, message, "未找到相关作品")
		return nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个相关作品:\n\n", len(artworks)))
	for i, aw := range artworks {
		if aw == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a>\n", i+1, aw.SourceURL, utils.EscapeHTML(aw.Title)))
		if aw.R18 {
			sb.WriteString("(R18)\n")
		}
	}
	utils.ReplyMessage(ctx, message, sb.String())
	return nil
}
