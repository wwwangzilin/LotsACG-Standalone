package service

import (
	"context"
	"fmt"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
)

// R18Mode 是用户级 R18 过滤模式。
// on: 只看 R18; off: 只看全年龄; mixed: 两者都看 (默认)。
type R18Mode string

const (
	R18ModeOn    R18Mode = "on"
	R18ModeOff   R18Mode = "off"
	R18ModeMixed R18Mode = "mixed"
)

// r18ModeTTL 用户 R18 模式的有效期。
const r18ModeTTL = 30 * 24 * time.Hour

func r18ModeKey(userID int64) string {
	return fmt.Sprintf("r18mode:%d", userID)
}

// ValidR18Mode 校验模式值是否合法。
func ValidR18Mode(mode string) bool {
	switch R18Mode(mode) {
	case R18ModeOn, R18ModeOff, R18ModeMixed:
		return true
	}
	return false
}

// GetUserR18Mode 返回用户设置的 R18 模式; ok=false 表示用户从未设置。
// 未设置时默认 mixed。
func GetUserR18Mode(ctx context.Context, userID int64) (R18Mode, bool) {
	mode, err := kvstor.Get[string](ctx, r18ModeKey(userID))
	if err != nil || !ValidR18Mode(mode) {
		return R18ModeMixed, false
	}
	return R18Mode(mode), true
}

// SetUserR18Mode 设置用户的 R18 模式。
func SetUserR18Mode(ctx context.Context, userID int64, mode R18Mode) error {
	return kvstor.SetWithTTL(ctx, r18ModeKey(userID), string(mode), r18ModeTTL)
}

// R18ModeToPixivMode 将 R18 模式转换为 Pixiv 搜索的 mode 参数。
// all: 全部; safe: 全年龄; r18: 仅 R18。
func (m R18Mode) ToPixivMode() string {
	switch m {
	case R18ModeOn:
		return "r18"
	case R18ModeOff:
		return "safe"
	default:
		return "all"
	}
}
