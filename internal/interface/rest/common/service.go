package common

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

const (
	StateKeyService     = "serv"
	StateKeyConfig      = "cfg"
	StateKeyLogger      = "logger"
	StateKeyTelegramBot = "telegrambot" // DO NOT USE MustGetState to get this value!
)

// TelegramBotStatus 描述 Telegram bot 的运行时状态, 供设置页展示。
type TelegramBotStatus struct {
	BotUsername  string  `json:"bot_username"`
	ChannelID    int64   `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	GroupID      int64   `json:"group_id"`
	R18ChannelID int64   `json:"r18_channel_id"`
	Running      bool    `json:"running"`
	AllowedUsers []int64 `json:"allowed_users"`
}

type TelegramBot interface {
	SendArtworkInfo(ctx context.Context, sourceUrl string, chatID int64, appendCaption string)
	// PostArtworkToChannel 将指定来源链接的作品发布到主频道 (供 XP-Pusher 等外部调用)。
	PostArtworkToChannel(ctx context.Context, sourceURL string) error
	// Status 返回 bot 运行状态 (用户名、频道/群信息等)。
	Status(ctx context.Context) TelegramBotStatus
}

func GetState[T any](ctx fiber.Ctx, key string) (T, bool) {
	val, ok := ctx.App().State().Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	vv, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return vv, true
}

func MustGetState[T any](ctx fiber.Ctx, key string) T {
	val := ctx.App().State().MustGet(key)
	v, ok := val.(T)
	if !ok {
		panic("state: dependency type assertion failed!")
	}
	return v
}
