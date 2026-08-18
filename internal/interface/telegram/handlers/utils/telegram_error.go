package utils

import (
	"errors"
	"strings"

	"github.com/mymmrac/telego/telegoapi"
)

// IsFileTooBigError 判断错误是否为 Telegram Bot API 的 "file is too big"。
// 兼容被 oops 等包装过的错误。
func IsFileTooBigError(err error) bool {
	if err == nil {
		return false
	}
	var te *telegoapi.Error
	if errors.As(err, &te) {
		if strings.Contains(strings.ToLower(te.Description), "file is too big") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "file is too big")
}
