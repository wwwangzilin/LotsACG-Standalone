package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/kmua"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

// KMua 管理内嵌的 kmua-bot (Telegram 群聊机器人, Python) 进程。
// kmua-bot 源码已内嵌进 exe, 无需单独部署; 逻辑 100% 由原始 Python 项目负责。
//
//	/kmua             查看状态
//	/kmua start       启动 (首次会自动提取源码/uv 装依赖/生成 settings.toml)
//	/kmua stop        停止
//	/kmua restart     重启
//	/kmua log [n]     显示最近 n 行日志 (默认 30)
func KMua(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionSudo) {
		utils.ReplyMessage(ctx, message, "你没有执行此操作的权限")
		return nil
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	mgr, err := kmua.NewManager(runtimecfg.Get().KMua)
	if err != nil {
		utils.ReplyMessage(ctx, message, "初始化失败: "+err.Error())
		return nil
	}

	switch sub {
	case "start":
		statusMsg, _ := utils.ReplyMessage(ctx, message, "⏳ 正在准备 kmua-bot 环境...")
		progress := func(text string) {
			utils.EditMessage(ctx, statusMsg, text)
		}
		pid, err := mgr.Start(ctx, progress)
		if err != nil {
			utils.EditMessage(ctx, statusMsg, "❌ 启动失败: "+err.Error())
			return nil
		}
		utils.EditMessage(ctx, statusMsg,
			fmt.Sprintf("✅ kmua-bot 已启动 (PID %d)\n日志: <code>%s</code>\n逻辑由原始 Python 项目负责, 与本 bot 相互独立。", pid, mgr.LogPath()))
		return nil
	case "stop":
		pid, err := mgr.Stop(ctx)
		if err != nil {
			utils.ReplyMessage(ctx, message, "停止失败: "+err.Error())
			return nil
		}
		if pid == 0 {
			utils.ReplyMessage(ctx, message, "kmua-bot 未在运行")
		} else {
			utils.ReplyMessage(ctx, message, fmt.Sprintf("✅ 已停止 kmua-bot (PID %d)", pid))
		}
		return nil
	case "restart":
		_, _ = mgr.Stop(ctx)
		time.Sleep(time.Second)
		statusMsg, _ := utils.ReplyMessage(ctx, message, "⏳ 正在重启 kmua-bot...")
		progress := func(text string) {
			utils.EditMessage(ctx, statusMsg, text)
		}
		pid, err := mgr.Start(ctx, progress)
		if err != nil {
			utils.EditMessage(ctx, statusMsg, "❌ 重启失败: "+err.Error())
			return nil
		}
		utils.EditMessage(ctx, statusMsg, fmt.Sprintf("✅ kmua-bot 已重启 (PID %d)", pid))
		return nil
	case "log":
		n := 30
		if len(args) > 1 {
			if v, err := strconv.Atoi(args[1]); err == nil && v > 0 && v <= 500 {
				n = v
			}
		}
		tail := mgr.LogTail(n)
		if len(tail) > 3500 {
			tail = "..." + tail[len(tail)-3500:]
		}
		utils.ReplyMessage(ctx, message, "📄 kmua-bot 日志 (最近 "+strconv.Itoa(n)+" 行):\n<pre>"+escapeHTML(tail)+"</pre>")
		return nil
	default:
		pid := mgr.RunningPID(ctx)
		if pid == 0 {
			utils.ReplyMessage(ctx, message, "kmua-bot: 未在运行\n\n用法:\n/kmua start - 启动 (首次自动提取源码/uv 装依赖/生成 settings.toml)\n/kmua stop - 停止\n/kmua restart - 重启\n/kmua log - 查看日志")
			return nil
		}
		utils.ReplyMessage(ctx, message, fmt.Sprintf("kmua-bot: 正在运行 (PID %d)\n日志: <code>%s</code>\n\n/kmua stop 停止 | /kmua restart 重启 | /kmua log 日志", pid, mgr.LogPath()))
		return nil
	}
}
