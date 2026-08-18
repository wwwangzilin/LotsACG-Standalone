package handlers

import (
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
)

// R18ModeCmd 处理 /r18mode on|off|mixed 指令:
// 设置当前用户的 R18 过滤模式, 影响随机图(/random /setu)和推荐(/recommend)。
// 不带参数时显示当前状态。
func R18ModeCmd(ctx *telegohandler.Context, message telego.Message) error {
	_, _, args := telegoutil.ParseCommand(message.Text)
	modeStr := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))

	if modeStr == "" {
		cur, _ := service.GetUserR18Mode(ctx, message.From.ID)
		desc := map[service.R18Mode]string{
			service.R18ModeOn:    "只看 R18",
			service.R18ModeOff:   "只看全年龄",
			service.R18ModeMixed: "全部",
		}[cur]
		utils.ReplyMessage(ctx, message, fmt.Sprintf("当前 R18 模式: <code>%s</code> (%s)\n\n用法: /r18mode on|off|mixed\n- on: 只看 R18\n- off: 只看全年龄\n- mixed: 全部(默认)", cur, desc))
		return nil
	}

	if !service.ValidR18Mode(modeStr) {
		utils.ReplyMessage(ctx, message, "无效的模式, 可选: on / off / mixed\n用法: /r18mode on|off|mixed")
		return nil
	}

	if err := service.SetUserR18Mode(ctx, message.From.ID, service.R18Mode(modeStr)); err != nil {
		utils.ReplyMessage(ctx, message, "设置失败, 请稍后再试")
		return err
	}
	desc := map[string]string{
		"on":    "只看 R18",
		"off":   "只看全年龄",
		"mixed": "全部(默认)",
	}[modeStr]
	utils.ReplyMessage(ctx, message, fmt.Sprintf("R18 模式已设置为 <code>%s</code> (%s)", modeStr, desc))
	return nil
}
