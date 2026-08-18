package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

// Channel 管理发送频道 (除主频道外的额外发送目标)。
// 作品发布时按各频道的规则自动推送 (不落库); 可通过命令或配置文件添加。
//
//	/channel                         查看帮助
//	/channel list                    列出所有发送频道
//	/channel add [选项] [chatID|@username] [标题]   添加 (在群/频道里可省略 chatID, 使用当前聊天)
//	/channel remove <chatID|@username>
//	/channel enable <chatID|@username>
//	/channel disable <chatID|@username>
//	/channel rule <chatID|@username> [选项]   修改发布规则
//
//	选项:
//	  -r18=follow|allow|deny|only    R18 规则 (follow=跟随, allow=允许, deny=拒绝, only=仅R18)
//	  -link / -nolink                是否只发链接 (不下载/上传图片)
//	  -t=标签1,标签2                 标签白名单 (任一命中即发布)
//	  -x=标签1,标签2                 标签黑名单 (任一命中即拒绝)
//	  -at=画师1,画师2                画师白名单
//	  -ax=画师1,画师2                画师黑名单
//	  -title=新标题                   设置标题
func Channel(ctx *telegohandler.Context, message telego.Message) error {
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

	switch sub {
	case "list":
		return channelList(ctx, message, serv)
	case "add":
		return channelAdd(ctx, message, serv, args[1:])
	case "remove", "del":
		return channelRemove(ctx, message, serv, args[1:])
	case "enable", "disable":
		return channelToggle(ctx, message, serv, args)
	case "rule", "set":
		return channelRule(ctx, message, serv, args[1:])
	default:
		utils.ReplyMessage(ctx, message, channelHelp())
		return nil
	}
}

func channelHelp() string {
	return `📡 发送频道管理

除主频道外, 可添加多个发送频道 (群/频道), 作品发布时按各频道规则自动推送。

用法:
/channel list                       列出所有发送频道
/channel add [选项] [chatID|@username] [标题]   添加频道 (在群/频道里可不带 chatID, 用当前聊天)
/channel remove <chatID|@username>  删除频道
/channel enable <chatID|@username>  启用频道
/channel disable <chatID|@username> 停用频道
/channel rule <chatID|@username> [选项]   修改发布规则

选项:
-r18=follow|allow|deny|only   R18 规则 (follow 跟随 / allow 允许 / deny 拒绝 / only 仅 R18)
-link / -nolink               只发链接 (不下载/上传图片) / 恢复正常发布
-t=标签1,标签2                标签白名单 (任一命中即发布)
-x=标签1,标签2                标签黑名单 (任一命中即拒绝)
-at=画师1,画师2               画师白名单
-ax=画师1,画师2               画师黑名单
-title=新标题                 设置标题

示例:
/channel add @my_channel 我的频道
/channel add -r18=deny -x=scat @safe_channel 安全频道
/channel rule @my_channel -link -r18=only`
}

func channelList(ctx *telegohandler.Context, message telego.Message, serv *service.Service) error {
	chs, err := serv.ListSendChannels(ctx)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取发送频道失败: "+err.Error())
		return nil
	}
	if len(chs) == 0 {
		utils.ReplyMessage(ctx, message, "📡 暂无发送频道\n\n使用 /channel add 添加, 或在 config.toml 的 [telegram.send_channels] 中配置。")
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📡 发送频道 (%d):\n", len(chs))
	for i, ch := range chs {
		fmt.Fprintf(&b, "\n%d. <b>%s</b> (<code>%d</code>)\n", i+1, escapeHTML(ch.Title), ch.ChatID)
		if ch.Enabled {
			b.WriteString("   状态: ✅ 启用\n")
		} else {
			b.WriteString("   状态: ⛔ 停用\n")
		}
		if ch.R18Mode != "" && ch.R18Mode != service.SendChannelR18Follow {
			fmt.Fprintf(&b, "   R18: %s\n", ch.R18Mode)
		}
		if ch.LinkOnly {
			b.WriteString("   仅链接: ✅\n")
		}
		if len(ch.IncludeTags) > 0 {
			fmt.Fprintf(&b, "   标签白名单: %s\n", strings.Join(ch.IncludeTags, ", "))
		}
		if len(ch.ExcludeTags) > 0 {
			fmt.Fprintf(&b, "   标签黑名单: %s\n", strings.Join(ch.ExcludeTags, ", "))
		}
		if len(ch.IncludeArtists) > 0 {
			fmt.Fprintf(&b, "   画师白名单: %s\n", strings.Join(ch.IncludeArtists, ", "))
		}
		if len(ch.ExcludeArtists) > 0 {
			fmt.Fprintf(&b, "   画师黑名单: %s\n", strings.Join(ch.ExcludeArtists, ", "))
		}
	}
	if len(b.String()) > 4000 {
		utils.ReplyMessage(ctx, message, b.String()[:4000])
		return nil
	}
	utils.ReplyMessage(ctx, message, b.String())
	return nil
}

func channelAdd(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	opts, rest, err := parseChannelOpts(args)
	if err != nil {
		utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
		return nil
	}
	var (
		chatID int64
		title  string
	)
	if len(rest) == 0 {
		// 在群/频道里用当前聊天
		if message.Chat.ChatID().ID == 0 {
			utils.ReplyMessage(ctx, message, "无法确定目标聊天, 请提供 chatID 或 @username")
			return nil
		}
		chatID = message.Chat.ChatID().ID
	} else {
		// 第一个参数是 chatID / @username, 剩余是标题
		id, err := resolveChatID(ctx, rest[0])
		if err != nil {
			// 可能是标题 (在群里 /channel add 标题)
			if message.Chat.ChatID().ID != 0 {
				chatID = message.Chat.ChatID().ID
				title = strings.Join(rest, " ")
			} else {
				utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
				return nil
			}
		} else {
			chatID = id
			title = strings.Join(rest[1:], " ")
		}
	}
	if opts.title != "" {
		title = opts.title
	}
	ch := &service.SendChannel{
		ChatID:  chatID,
		Title:   title,
		Enabled: true,
	}
	applyOpts(ch, opts)
	if err := serv.AddSendChannel(ctx, ch); err != nil {
		utils.ReplyMessage(ctx, message, "添加发送频道失败: "+err.Error())
		return nil
	}
	utils.ReplyMessage(ctx, message,
		fmt.Sprintf("✅ 已添加发送频道:\n%s", formatChannel(ch)))
	return nil
}

func channelRemove(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	if len(args) == 0 {
		utils.ReplyMessage(ctx, message, "用法: /channel remove <chatID|@username>")
		return nil
	}
	chatID, err := resolveChatID(ctx, args[0])
	if err != nil {
		utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
		return nil
	}
	if err := serv.RemoveSendChannel(ctx, chatID); err != nil {
		utils.ReplyMessage(ctx, message, "删除失败: "+err.Error())
		return nil
	}
	utils.ReplyMessage(ctx, message, fmt.Sprintf("🗑️ 已删除发送频道 <code>%d</code>", chatID))
	return nil
}

func channelToggle(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	if len(args) < 2 {
		utils.ReplyMessage(ctx, message, "用法: /channel enable|disable <chatID|@username>")
		return nil
	}
	chatID, err := resolveChatID(ctx, args[1])
	if err != nil {
		utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
		return nil
	}
	ch, err := serv.GetSendChannel(ctx, chatID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "频道不存在: "+err.Error())
		return nil
	}
	ch.Enabled = strings.EqualFold(args[0], "enable")
	if err := serv.SaveSendChannel(ctx, ch); err != nil {
		utils.ReplyMessage(ctx, message, "操作失败: "+err.Error())
		return nil
	}
	state := "✅ 已启用"
	if !ch.Enabled {
		state = "⛔ 已停用"
	}
	utils.ReplyMessage(ctx, message, fmt.Sprintf("%s发送频道 <code>%d</code>", state, chatID))
	return nil
}

func channelRule(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	if len(args) == 0 {
		utils.ReplyMessage(ctx, message, "用法: /channel rule <chatID|@username> [选项]")
		return nil
	}
	chatID, err := resolveChatID(ctx, args[0])
	if err != nil {
		utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
		return nil
	}
	ch, err := serv.GetSendChannel(ctx, chatID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "频道不存在: "+err.Error())
		return nil
	}
	opts, rest, err := parseChannelOpts(args[1:])
	if err != nil {
		utils.ReplyMessage(ctx, message, "参数错误: "+err.Error())
		return nil
	}
	if opts.title != "" {
		ch.Title = opts.title
	}
	if len(rest) > 0 && opts.title == "" {
		// /channel rule <id> 新标题
		ch.Title = strings.Join(rest, " ")
	}
	applyOpts(ch, opts)
	if err := serv.SaveSendChannel(ctx, ch); err != nil {
		utils.ReplyMessage(ctx, message, "更新失败: "+err.Error())
		return nil
	}
	utils.ReplyMessage(ctx, message, fmt.Sprintf("✅ 已更新发送频道规则:\n%s", formatChannel(ch)))
	return nil
}

// channelOpts 由命令选项解析出的规则更新。
type channelOpts struct {
	r18         string
	linkOnly    *bool
	title       string
	includeTags []string
	excludeTags []string
	includeArt  []string
	excludeArt  []string
	// setXXX 区分「未设置」和「显式清空(-t=)」
	setIncludeTags bool
	setExcludeTags bool
	setIncludeArt  bool
	setExcludeArt  bool
}

// parseChannelOpts 解析 `-key=value` / `-flag` 形式的选项, 返回剩余位置参数。
func parseChannelOpts(args []string) (*channelOpts, []string, error) {
	opts := &channelOpts{}
	var rest []string
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "-r18="):
			v := strings.ToLower(strings.TrimPrefix(a, "-r18="))
			switch v {
			case service.SendChannelR18Follow, service.SendChannelR18Allow, service.SendChannelR18Deny, service.SendChannelR18Only:
				opts.r18 = v
			default:
				return nil, nil, oops.Errorf("无效的 -r18 值: %s (可选 follow|allow|deny|only)", v)
			}
		case a == "-link":
			v := true
			opts.linkOnly = &v
		case a == "-nolink":
			v := false
			opts.linkOnly = &v
		case strings.HasPrefix(a, "-t="):
			opts.includeTags = splitCommaList(strings.TrimPrefix(a, "-t="))
			opts.setIncludeTags = true
		case strings.HasPrefix(a, "-x="):
			opts.excludeTags = splitCommaList(strings.TrimPrefix(a, "-x="))
			opts.setExcludeTags = true
		case strings.HasPrefix(a, "-at="):
			opts.includeArt = splitCommaList(strings.TrimPrefix(a, "-at="))
			opts.setIncludeArt = true
		case strings.HasPrefix(a, "-ax="):
			opts.excludeArt = splitCommaList(strings.TrimPrefix(a, "-ax="))
			opts.setExcludeArt = true
		case strings.HasPrefix(a, "-title="):
			opts.title = strings.TrimPrefix(a, "-title=")
		case strings.HasPrefix(a, "-"):
			return nil, nil, oops.Errorf("未知选项: %s", a)
		default:
			rest = append(rest, a)
		}
	}
	return opts, rest, nil
}

// applyOpts 把解析出的选项应用到频道。未设置的字段保持原值。
func applyOpts(ch *service.SendChannel, opts *channelOpts) {
	if opts.r18 != "" {
		ch.R18Mode = opts.r18
	}
	if opts.linkOnly != nil {
		ch.LinkOnly = *opts.linkOnly
	}
	if opts.setIncludeTags {
		ch.IncludeTags = opts.includeTags
	}
	if opts.setExcludeTags {
		ch.ExcludeTags = opts.excludeTags
	}
	if opts.setIncludeArt {
		ch.IncludeArtists = opts.includeArt
	}
	if opts.setExcludeArt {
		ch.ExcludeArtists = opts.excludeArt
	}
}

func splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// resolveChatID 把命令行参数解析为数字 chatID; 支持 -100xxx 或 @username (通过 getChat 解析)。
func resolveChatID(ctx *telegohandler.Context, arg string) (int64, error) {
	arg = strings.TrimSpace(arg)
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return id, nil
	}
	u := strings.TrimPrefix(arg, "@")
	if u == "" {
		return 0, oops.New("聊天 ID 为空")
	}
	chat, err := ctx.Bot().GetChat(ctx, &telego.GetChatParams{ChatID: telegoutil.Username(u)})
	if err != nil {
		return 0, oops.Errorf("无法解析 %q: 请确认 bot 已加入该群/频道 (%v)", arg, err)
	}
	return chat.ID, nil
}

func formatChannel(ch *service.SendChannel) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<b>%s</b> (<code>%d</code>)\n", escapeHTML(ch.Title), ch.ChatID)
	if ch.Enabled {
		b.WriteString("状态: ✅ 启用\n")
	} else {
		b.WriteString("状态: ⛔ 停用\n")
	}
	if ch.R18Mode != "" {
		fmt.Fprintf(&b, "R18: %s\n", ch.R18Mode)
	}
	b.WriteString(fmt.Sprintf("仅链接: %v\n", ch.LinkOnly))
	if len(ch.IncludeTags) > 0 {
		fmt.Fprintf(&b, "标签白名单: %s\n", strings.Join(ch.IncludeTags, ", "))
	}
	if len(ch.ExcludeTags) > 0 {
		fmt.Fprintf(&b, "标签黑名单: %s\n", strings.Join(ch.ExcludeTags, ", "))
	}
	if len(ch.IncludeArtists) > 0 {
		fmt.Fprintf(&b, "画师白名单: %s\n", strings.Join(ch.IncludeArtists, ", "))
	}
	if len(ch.ExcludeArtists) > 0 {
		fmt.Fprintf(&b, "画师黑名单: %s\n", strings.Join(ch.ExcludeArtists, ", "))
	}
	return strings.TrimRight(b.String(), "\n")
}
