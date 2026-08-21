package handlers

import (
	"fmt"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/version"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func Help(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	helpText := `使用方法:
/start - 开始使用
/help - 显示本帮助
/setu - 随机图片(NSFW)
/random - 随机全年龄图片
/search - 搜索相似图片
/info - 发送作品图片和信息
/files (或 /file) - 获取作品原图
/hybrid - 混合搜索作品
/similar - 搜索相似作品
/status - 查看机器人状态 (作品数/队列/版本等)
/r18mode - 设置 R18 过滤模式 (on/off/mixed)
/groupsearch - 在群组中搜索作品
/downloadzip - 打包下载回复图片及之后的多张图片 (zip)
/tagging - 识别回复图片中的标签
/follow - 关注画师, 画师新作品自动推送 (支持 -g=组名 归组)
/unfollow - 取消关注画师 (支持多个链接或 -g=组名 批量)
/followlist - 查看关注的画师 (支持 -p=页 -platform=平台 -g=组名)
/followgroup - 关注分组管理 (list/add/del)
/sub - 订阅标签, 有新作品时自动推送
/unsub - 取消订阅标签
/sublist - 查看订阅的标签
`
	helpText += `
随机图片相关功能中支持使用以下格式的参数:
使用 '|' 分隔'或'关系, 使用 '空格' 分隔'与'关系, 示例:

/random 萝莉|白丝 猫耳|原创

表示搜索包含"萝莉"或"白丝", 且包含"猫耳"或"原创"的图片
Inline 查询(在任意聊天框中@本bot)支持同样的参数格式
`
	isAdmin, _ := serv.IsAdminByTgID(ctx, message.From.ID)
	if isAdmin {
		helpText += `
管理员命令:
/addadmin - 添加管理员
/deladmin - 删除管理员
/delete (或 /del) - 删除整个作品
/r18 - 设置作品R18标记
/title - 设置作品标题
/tags - 更新作品标签(覆盖原有标签)
/addtags - 添加作品标签
/deltags - 删除作品标签
/post - 将作品发布到频道(支持链接或回复消息)
/refresh - 刷新作品缓存和文件信息
/tagalias - 为标签添加别名
/autotag - 自动tag作品
/dump - 输出 json 格式作品信息
/recaption - 重新生成作品描述
/reindex - 重新索引作品
/dupcheck - 查看或切换图片查重开关 (on/off)
/cancel - 取消当前发布队列 (保留已发布内容)
/cd - 取消并删除当前发布队列已发布的内容
/redescribe - 使用 AI 生成描述并追加到频道帖子下方
/update - 检查并自动更新到最新版本
/xppusher - 管理 XP-Pusher (Python) 进程: start/stop/restart/status/key
/kmua - 管理 kmua-bot (Python) 进程: start/stop/restart/status/log
`
	}
	commit := version.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	helpText += fmt.Sprintf("\n版本: %s, 构建日期 %s, 提交 %s\nhttps://github.com/wwwangzilin/LotsACG-Standalone", version.Version, version.BuildTime, commit)
	utils.ReplyMessage(ctx, message, helpText)
	return nil
}
