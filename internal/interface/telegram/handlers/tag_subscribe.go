package handlers

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
)

// SubscribeTag 处理 /sub: 订阅一个标签, 有该标签的新作品时自动推送。
func SubscribeTag(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	tag := strings.TrimSpace(strings.Join(args, " "))
	if tag == "" {
		utils.ReplyMessage(ctx, message, "请提供要订阅的标签, 例如:\n/sub 原神\n/sub 白丝")
		return nil
	}
	added, err := serv.SubscribeTag(ctx, message.From.ID, tag)
	if err != nil {
		utils.ReplyMessage(ctx, message, "订阅失败: "+err.Error())
		return nil
	}
	if added {
		utils.ReplyMessage(ctx, message, "✅ 已订阅标签「"+tag+"」\n有该标签的新作品时会自动推送给你\n使用 /unsub 取消订阅")
	} else {
		utils.ReplyMessage(ctx, message, "你已经订阅过该标签了")
	}
	return nil
}

// UnsubscribeTag 处理 /unsub: 取消订阅一个标签。
func UnsubscribeTag(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	tag := strings.TrimSpace(strings.Join(args, " "))
	if tag == "" {
		utils.ReplyMessage(ctx, message, "请提供要取消订阅的标签, 例如:\n/unsub 原神")
		return nil
	}
	removed, err := serv.UnsubscribeTag(ctx, message.From.ID, tag)
	if err != nil {
		utils.ReplyMessage(ctx, message, "取消失败: "+err.Error())
		return nil
	}
	if removed {
		utils.ReplyMessage(ctx, message, "已取消订阅标签「"+tag+"」")
	} else {
		utils.ReplyMessage(ctx, message, "你还没有订阅该标签")
	}
	return nil
}

// TagSubList 处理 /sublist: 查看当前订阅的标签。
func TagSubList(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	tags, err := serv.ListSubscribedTags(ctx, message.From.ID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取订阅列表失败: "+err.Error())
		return nil
	}
	if len(tags) == 0 {
		utils.ReplyMessage(ctx, message, "你还没有订阅任何标签\n使用 /sub <标签> 订阅")
		return nil
	}
	utils.ReplyMessage(ctx, message, "你订阅的标签 ("+strconv.Itoa(len(tags))+"):\n"+strings.Join(tags, "\n"))
	return nil
}
