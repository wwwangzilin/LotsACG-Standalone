package service

import (
	"context"
	"strconv"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
)

const dupCheckKey = "dupcheck:enabled"

// GetDupCheckEnabled returns whether image duplicate checking is enabled.
// KV store value (set by /dupcheck command) takes precedence, otherwise the
// config default is used.
func GetDupCheckEnabled(ctx context.Context) bool {
	if v, err := kvstor.Get[string](ctx, dupCheckKey); err == nil {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return runtimecfg.Get().Search.DupCheckEnable
}

// SetDupCheckEnabled persists the duplicate checking toggle to the KV store.
func SetDupCheckEnabled(ctx context.Context, enabled bool) error {
	return kvstor.Set(ctx, dupCheckKey, strconv.FormatBool(enabled))
}
