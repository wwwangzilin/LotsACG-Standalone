package handlers

import (
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
)

// ToggleDupCheck 切换图片查重开关
//
// 语法:
// /dupcheck      - 查看当前状态
// /dupcheck on   - 开启查重
// /dupcheck off  - 关闭查重
func ToggleDupCheck(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionPostArtwork) {
		utils.ReplyMessage(ctx, message, "你没有权限管理查重开关")
		return nil
	}

	_, _, args := telegoutil.ParseCommand(message.Text)

	if len(args) == 0 {
		enabled := service.GetDupCheckEnabled(ctx)
		status := "开启"
		if !enabled {
			status = "关闭"
		}
		utils.ReplyMessage(ctx, message, "图片查重当前状态: "+status)
		return nil
	}

	switch args[0] {
	case "on", "开启", "1", "true":
		if err := service.SetDupCheckEnabled(ctx, true); err != nil {
			utils.ReplyMessage(ctx, message, "设置失败: "+err.Error())
			return nil
		}
		utils.ReplyMessage(ctx, message, "已开启图片查重")
	case "off", "关闭", "0", "false":
		if err := service.SetDupCheckEnabled(ctx, false); err != nil {
			utils.ReplyMessage(ctx, message, "设置失败: "+err.Error())
			return nil
		}
		utils.ReplyMessage(ctx, message, "已关闭图片查重")
	default:
		utils.ReplyMessage(ctx, message, "用法: /dupcheck [on|off]")
	}
	return nil
}
