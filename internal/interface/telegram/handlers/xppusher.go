package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/unvgo/ouid"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/xppusher"
)

// XPPusher 管理 Pixiv-XP-Pusher (Python) 进程。
// XP-Pusher 源码已内嵌进 exe, 无需单独部署; 推荐逻辑 100% 由原始 Python 项目负责。
//
//	/xppusher             查看状态
//	/xppusher start       启动 (首次会自动提取源码/建 venv/装依赖)
//	/xppusher stop        停止
//	/xppusher restart     重启
//	/xppusher log [n]     显示最近 n 行日志 (默认 30)
//	/xppusher key         生成/显示推送到群用的 API Key
func XPPusher(ctx *telegohandler.Context, message telego.Message) error {
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

	mgr, err := xppusher.NewManager(runtimecfg.Get().XPPusher)
	if err != nil {
		utils.ReplyMessage(ctx, message, "初始化失败: "+err.Error())
		return nil
	}

	switch sub {
	case "start":
		statusMsg, _ := utils.ReplyMessage(ctx, message, "⏳ 正在准备 XP-Pusher 环境...")
		progress := func(text string) {
			utils.EditMessage(ctx, statusMsg, text)
		}
		pid, err := mgr.Start(ctx, progress)
		if err != nil {
			utils.EditMessage(ctx, statusMsg, "❌ 启动失败: "+err.Error())
			return nil
		}
		utils.EditMessage(ctx, statusMsg,
			fmt.Sprintf("✅ XP-Pusher 已启动 (PID %d)\n日志: <code>%s</code>\n推荐逻辑由原始 Python 项目负责, 与本 bot 相互独立。", pid, mgr.LogPath()))
		return nil
	case "stop":
		pid, err := mgr.Stop(ctx)
		if err != nil {
			utils.ReplyMessage(ctx, message, "停止失败: "+err.Error())
			return nil
		}
		if pid == 0 {
			utils.ReplyMessage(ctx, message, "XP-Pusher 未在运行")
		} else {
			utils.ReplyMessage(ctx, message, fmt.Sprintf("✅ 已停止 XP-Pusher (PID %d)", pid))
		}
		return nil
	case "restart":
		_, _ = mgr.Stop(ctx)
		time.Sleep(time.Second)
		statusMsg, _ := utils.ReplyMessage(ctx, message, "⏳ 正在重启 XP-Pusher...")
		progress := func(text string) {
			utils.EditMessage(ctx, statusMsg, text)
		}
		pid, err := mgr.Start(ctx, progress)
		if err != nil {
			utils.EditMessage(ctx, statusMsg, "❌ 重启失败: "+err.Error())
			return nil
		}
		utils.EditMessage(ctx, statusMsg, fmt.Sprintf("✅ XP-Pusher 已重启 (PID %d)", pid))
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
		utils.ReplyMessage(ctx, message, "📄 XP-Pusher 日志 (最近 "+strconv.Itoa(n)+" 行):\n<pre>"+escapeHTML(tail)+"</pre>")
		return nil
	case "key":
		key, err := createXPPushKey(ctx, serv)
		if err != nil {
			utils.ReplyMessage(ctx, message, "生成 API Key 失败: "+err.Error())
			return nil
		}
		utils.ReplyMessage(ctx, message,
			fmt.Sprintf("🔑 已生成「推送到群」API Key:\n<code>%s</code>\n\n请在 XP-Pusher 的 config.yaml 中添加:\n<code>lotsacg:\n  url: \"http://127.0.0.1:%s\"\n  api_key: \"%s\"</code>",
				key, restPort(), key))
		return nil
	default:
		pid := mgr.RunningPID(ctx)
		if pid == 0 {
			utils.ReplyMessage(ctx, message, "XP-Pusher: 未在运行\n\n用法:\n/xppusher start - 启动 (首次自动提取源码/建 venv/装依赖)\n/xppusher stop - 停止\n/xppusher restart - 重启\n/xppusher log - 查看日志\n/xppusher key - 生成推送到群用的 API Key")
			return nil
		}
		utils.ReplyMessage(ctx, message, fmt.Sprintf("XP-Pusher: 正在运行 (PID %d)\n日志: <code>%s</code>\n\n/xppusher stop 停止 | /xppusher restart 重启 | /xppusher log 日志", pid, mgr.LogPath()))
		return nil
	}
}

// createXPPushKey 生成一个带「发布作品」权限的 API Key, 供 XP-Pusher 推送到群使用。
func createXPPushKey(ctx context.Context, serv *service.Service) (string, error) {
	key := "lotsacg_" + ouid.New().Hex()
	if _, err := serv.CreateApiKey(ctx, key, 0, []shared.Permission{shared.PermissionPostArtwork}, "xppusher push"); err != nil {
		return "", err
	}
	return key, nil
}

// restPort 返回 REST 服务端口, 用于生成默认 URL。
func restPort() string {
	addr := runtimecfg.Get().Rest.Addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return "8080"
}

// escapeHTML 转义 HTML 特殊字符 (用于日志展示)。
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}
