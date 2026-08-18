package handlers

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
)

// FollowArtist 处理 /follow: 关注一个画师, 画师发布新作品时自动推送。
func FollowArtist(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	text := strings.Join(args, " ")
	artistURL := serv.FindArtistPageURL(text)
	if artistURL == "" {
		utils.ReplyMessage(ctx, message, "请提供有效的画师主页链接, 例如:\n/follow https://www.pixiv.net/users/123456")
		return nil
	}
	added, err := serv.FollowArtist(ctx, message.From.ID, artistURL)
	if err != nil {
		utils.ReplyMessage(ctx, message, "关注失败: "+err.Error())
		return nil
	}
	if added {
		utils.ReplyMessage(ctx, message, "✅ 已关注画师 "+artistURL+"\n画师发布新作品时会自动推送给你\n使用 /unfollow 取消关注")
	} else {
		utils.ReplyMessage(ctx, message, "你已经关注过该画师了")
	}
	return nil
}

// UnfollowArtist 处理 /unfollow: 取消关注一个画师。
func UnfollowArtist(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	text := strings.Join(args, " ")
	artistURL := serv.FindArtistPageURL(text)
	if artistURL == "" {
		utils.ReplyMessage(ctx, message, "请提供有效的画师主页链接, 例如:\n/unfollow https://www.pixiv.net/users/123456")
		return nil
	}
	removed, err := serv.UnfollowArtist(ctx, message.From.ID, artistURL)
	if err != nil {
		utils.ReplyMessage(ctx, message, "取消失败: "+err.Error())
		return nil
	}
	if removed {
		utils.ReplyMessage(ctx, message, "已取消关注画师 "+artistURL)
	} else {
		utils.ReplyMessage(ctx, message, "你还没有关注该画师")
	}
	return nil
}

// FollowList 处理 /followlist: 查看当前关注的画师。
func FollowList(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	urls, err := serv.ListFollowArtists(ctx, message.From.ID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取关注列表失败: "+err.Error())
		return nil
	}
	if len(urls) == 0 {
		utils.ReplyMessage(ctx, message, "你还没有关注任何画师\n使用 /follow <画师主页链接> 关注")
		return nil
	}
	utils.ReplyMessage(ctx, message, "你关注的画师 ("+strconv.Itoa(len(urls))+"):\n"+strings.Join(urls, "\n"))
	return nil
}
