package handlers

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
)

// followPageSize /followlist 每页显示的画师数量。
const followPageSize = 10

// parseFollowOpts 从命令参数中提取 `-key=value` 选项, 返回剩余位置参数。
// 支持: -g=组名 (关注分组) / -p=N (页码) / -platform=X (平台筛选)。
func parseFollowOpts(args []string) (map[string]string, []string) {
	opts := make(map[string]string)
	var rest []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") && strings.Contains(a, "=") {
			kv := strings.SplitN(a, "=", 2)
			opts[strings.TrimPrefix(kv[0], "-")] = strings.TrimSpace(kv[1])
		} else {
			rest = append(rest, a)
		}
	}
	return opts, rest
}

// FollowArtist 处理 /follow: 关注一个画师, 画师发布新作品时自动推送。
// 支持: /follow [-g=组名] <画师主页链接>  或  /follow --all (一键订阅所有已发布作者)
func FollowArtist(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	opts, rest := parseFollowOpts(args)

	// 一键订阅所有已发布过作品的作者
	for _, a := range args {
		if strings.EqualFold(a, "--all") || strings.EqualFold(a, "-all") {
			subscribed, err := serv.SubscribeAllArtists(ctx, message.From.ID)
			if err != nil {
				utils.ReplyMessage(ctx, message, "订阅失败: "+err.Error())
				return nil
			}
			utils.ReplyMessage(ctx, message, fmt.Sprintf("✅ 已一键订阅全部作者, 本次新增 %d 位\n订阅作者的新作品将自动补齐发布到主频道", subscribed))
			return nil
		}
	}

	text := strings.Join(rest, " ")
	artistURL := serv.FindArtistPageURL(text)
	if artistURL == "" {
		utils.ReplyMessage(ctx, message, "请提供有效的画师主页链接, 例如:\n/follow https://www.pixiv.net/users/123456\n/follow -g=佬 https://www.pixiv.net/users/123456\n一键订阅全部作者: /follow --all")
		return nil
	}
	added, err := serv.FollowArtistWithGroup(ctx, message.From.ID, artistURL, opts["g"])
	if err != nil {
		utils.ReplyMessage(ctx, message, "关注失败: "+err.Error())
		return nil
	}
	if added {
		msg := "✅ 已关注画师 " + artistURL + "\n画师发布新作品时会自动推送给你\n使用 /unfollow 取消关注"
		if g := opts["g"]; g != "" {
			msg += "\n已归入分组: " + g
		}
		utils.ReplyMessage(ctx, message, msg)
	} else {
		msg := "你已经关注过该画师了"
		if g := opts["g"]; g != "" {
			msg += "\n(已将其归入分组 " + g + ")"
		}
		utils.ReplyMessage(ctx, message, msg)
	}
	return nil
}

// UnfollowArtist 处理 /unfollow: 取消关注画师。
// 支持批量: /unfollow <url1> <url2> ...  或按组: /unfollow -g=组名
func UnfollowArtist(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	opts, rest := parseFollowOpts(args)

	// 按组批量取消
	if g := opts["g"]; g != "" {
		removed, err := serv.UnfollowArtistGroup(ctx, message.From.ID, g)
		if err != nil {
			utils.ReplyMessage(ctx, message, "操作失败: "+err.Error())
			return nil
		}
		utils.ReplyMessage(ctx, message, fmt.Sprintf("已取消关注分组 %q 下的 %d 个画师", g, removed))
		return nil
	}

	urls := serv.FindArtistPageURLs(strings.Join(rest, " "))
	if len(urls) == 0 {
		utils.ReplyMessage(ctx, message, "请提供有效的画师主页链接, 例如:\n/unfollow https://www.pixiv.net/users/123456\n支持多个链接批量取消: /unfollow url1 url2")
		return nil
	}
	removed, err := serv.UnfollowArtists(ctx, message.From.ID, urls)
	if err != nil {
		utils.ReplyMessage(ctx, message, "操作失败: "+err.Error())
		return nil
	}
	if removed == 0 {
		utils.ReplyMessage(ctx, message, "你还没有关注这些画师")
		return nil
	}
	utils.ReplyMessage(ctx, message, fmt.Sprintf("已取消关注 %d 个画师", removed))
	return nil
}

// FollowList 处理 /followlist: 查看当前关注的画师。
// 支持: /followlist [-p=页码] [-platform=pixiv|twitter] [-g=组名]
func FollowList(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	opts, _ := parseFollowOpts(args)

	page := 1
	if p := opts["p"]; p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	platform := strings.ToLower(opts["platform"])
	group := opts["g"]

	entries, err := serv.ListFollowArtistsDetail(ctx, message.From.ID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取关注列表失败: "+err.Error())
		return nil
	}
	// 按平台 / 分组筛选
	filtered := make([]service.FollowArtistEntry, 0, len(entries))
	for _, e := range entries {
		if platform != "" && e.Platform != platform {
			continue
		}
		if group != "" && !containsString(e.Groups, group) {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == 0 {
		hint := "你还没有关注的画师\n使用 /follow <画师主页链接> 关注"
		if platform != "" || group != "" {
			hint = "没有符合条件的关注画师"
		}
		utils.ReplyMessage(ctx, message, hint)
		return nil
	}
	totalPages := (len(filtered) + followPageSize - 1) / followPageSize
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * followPageSize
	end := start + followPageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	var b strings.Builder
	title := fmt.Sprintf("你关注的画师 (共 %d 个", len(filtered))
	if platform != "" {
		title += ", 平台: " + platform
	}
	if group != "" {
		title += ", 分组: " + group
	}
	title += ")"
	fmt.Fprintf(&b, "%s, 第 %d/%d 页:\n", title, page, totalPages)
	for i, e := range filtered[start:end] {
		fmt.Fprintf(&b, "\n%d. [%s] %s", start+i+1, e.Platform, e.URL)
		if len(e.Groups) > 0 {
			fmt.Fprintf(&b, "\n   分组: %s", strings.Join(e.Groups, ", "))
		}
	}
	if page < totalPages {
		fmt.Fprintf(&b, "\n\n使用 /followlist -p=%d 查看下一页", page+1)
	}
	utils.ReplyMessage(ctx, message, b.String())
	return nil
}

// FollowGroup 处理 /followgroup: 关注分组管理。
//
//	/followgroup list              列出所有分组及组内画师
//	/followgroup add <组名>         新建空分组
//	/followgroup del <组名>         删除分组 (画师仍保持关注)
func FollowGroup(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	switch sub {
	case "list":
		return followGroupList(ctx, message, serv)
	case "add":
		return followGroupAdd(ctx, message, serv, args[1:])
	case "del", "remove", "delete":
		return followGroupDel(ctx, message, serv, args[1:])
	default:
		utils.ReplyMessage(ctx, message, followGroupHelp())
		return nil
	}
}

func followGroupHelp() string {
	return `📁 关注分组管理

分组用于按收藏夹归类画师, 便于按组批量管理。
关注时用 /follow -g=组名 <画师链接> 归组。

用法:
/followgroup list              列出所有分组及组内画师
/followgroup add <组名>         新建空分组
/followgroup del <组名>         删除分组 (画师仍保持关注, 只是移出该组)

批量操作:
/unfollow -g=<组名>            取消关注该组全部画师
/followlist -g=<组名>           只看某个分组的画师`
}

func followGroupList(ctx *telegohandler.Context, message telego.Message, serv *service.Service) error {
	groups, err := serv.ListFollowGroups(ctx, message.From.ID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "获取分组失败: "+err.Error())
		return nil
	}
	if len(groups) == 0 {
		utils.ReplyMessage(ctx, message, "你还没有关注分组\n使用 /follow -g=组名 <画师链接> 关注并归组, 或 /followgroup add <组名> 新建空分组")
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📁 关注分组 (%d):\n", len(groups))
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		urls := groups[name]
		fmt.Fprintf(&b, "\n• <b>%s</b> (%d):\n", escapeHTML(name), len(urls))
		for _, u := range urls {
			fmt.Fprintf(&b, "  %s\n", u)
		}
	}
	utils.ReplyMessage(ctx, message, b.String())
	return nil
}

func followGroupAdd(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		utils.ReplyMessage(ctx, message, "用法: /followgroup add <组名>")
		return nil
	}
	groups, err := serv.ListFollowGroups(ctx, message.From.ID)
	if err != nil {
		utils.ReplyMessage(ctx, message, "操作失败: "+err.Error())
		return nil
	}
	if _, ok := groups[name]; ok {
		utils.ReplyMessage(ctx, message, "分组已存在: "+name)
		return nil
	}
	groups[name] = []string{}
	if err := serv.SaveFollowGroups(ctx, message.From.ID, groups); err != nil {
		utils.ReplyMessage(ctx, message, "操作失败: "+err.Error())
		return nil
	}
	utils.ReplyMessage(ctx, message, "✅ 已创建分组: "+name+"\n使用 /follow -g="+name+" <画师链接> 关注并归组")
	return nil
}

func followGroupDel(ctx *telegohandler.Context, message telego.Message, serv *service.Service, args []string) error {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		utils.ReplyMessage(ctx, message, "用法: /followgroup del <组名>")
		return nil
	}
	if err := serv.RemoveFollowGroup(ctx, message.From.ID, name); err != nil {
		utils.ReplyMessage(ctx, message, "删除失败: "+err.Error())
		return nil
	}
	utils.ReplyMessage(ctx, message, "🗑️ 已删除分组 "+name+" (组内画师仍保持关注)")
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
