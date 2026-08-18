package telegram

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/mymmrac/telego"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
)

var (
	CommonCommands = []telego.BotCommand{
		{
			Command:     "start",
			Description: "开始使用",
		},
		{
			Command:     "files",
			Description: "获取作品原图文件",
		},
		{
			Command:     "setu",
			Description: "来点涩图(R18)",
		},
		{
			Command:     "random",
			Description: "随机1张全年龄图片",
		},
		{
			Command:     "search",
			Description: "搜索相似图片",
		},
		{
			Command:     "info",
			Description: "获取作品图片和信息",
		},
		{
			Command:     "help",
			Description: "食用指南",
		},
		{
			Command:     "hybrid",
			Description: "基于语义与关键字混合搜索作品",
		},
		{
			Command:     "similar",
			Description: "获取与回复的图片相似的作品",
		},
		{
			Command:     "tagging",
			Description: "识别图片中的标签",
		},
		{
			Command:     "r18mode",
			Description: "设置 R18 过滤模式 (on/off/mixed)",
		},
		{
			Command:     "status",
			Description: "查看机器人状态",
		},
		{
			Command:     "groupsearch",
			Description: "在群组中搜索作品",
		},
		{
			Command:     "downloadzip",
			Description: "打包下载图片 (zip)",
		},
		{
			Command:     "follow",
			Description: "关注画师, 新作品自动推送",
		},
		{
			Command:     "unfollow",
			Description: "取消关注画师",
		},
		{
			Command:     "followlist",
			Description: "查看关注的画师",
		},
		{
			Command:     "sub",
			Description: "订阅标签, 新作品自动推送",
		},
		{
			Command:     "unsub",
			Description: "取消订阅标签",
		},
		{
			Command:     "sublist",
			Description: "查看订阅的标签",
		},
	}

	AdminCommands = []telego.BotCommand{
		{
			Command:     "addadmin",
			Description: "添加管理员",
		},
		{
			Command:     "deladmin",
			Description: "删除管理员",
		},
		{
			Command:     "delete",
			Description: "删除作品",
		},
		{
			Command:     "r18",
			Description: "更改作品 R18",
		},
		{
			Command:     "title",
			Description: "设置作品标题",
		},
		{
			Command:     "tags",
			Description: "设置作品标签(覆盖)",
		},
		{
			Command:     "autotag",
			Description: "自动添加作品标签",
		},
		{
			Command:     "addtags",
			Description: "添加作品标签",
		},
		{
			Command:     "deltags",
			Description: "删除作品标签",
		},
		{
			Command:     "tagalias",
			Description: "为标签添加别名",
		},
		{
			Command:     "post",
			Description: "发布作品",
		},
		{
			Command:     "refresh",
			Description: "刷新作品缓存",
		},
		{
			Command:     "recaption",
			Description: "重新生成作品描述",
		},
		{
			Command:     "dump",
			Description: "导出作品信息",
		},
		{
			Command:     "cancel",
			Description: "取消当前发布队列",
		},
		{
			Command:     "cd",
			Description: "取消并删除发布队列",
		},
		{
			Command:     "redescribe",
			Description: "AI 生成描述并追加到帖子",
		},
		{
			Command:     "update",
			Description: "检查并自动更新到最新版本",
		},
		{
			Command:     "channel",
			Description: "管理发送频道 (add/list/remove/rule)",
		},
	}
)

func commandsSignature(cfg runtimecfg.TelegramConfig) (string, error) {
	data := struct {
		Common []telego.BotCommand
		Admin  []telego.BotCommand
		Cfg    runtimecfg.TelegramConfig
	}{
		Common: CommonCommands,
		Admin:  AdminCommands,
		Cfg:    cfg,
	}
	b, err := msgpack.Marshal(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
