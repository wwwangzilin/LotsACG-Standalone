package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/goccy/go-json"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/app"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/version"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/unvgo/ouid"
)

const banner = `
 _          _            ____   ____   ____ 
| |    ___ | |_    __ _ / ___| / ___| / ___|
| |   / _ \| __|  / _  | |     | |    | |  _
| |__| (_) | |_  | (_| | |___  | |___ | |_| |
|_____\___/ \__|  \__,_|\____| \____| \____|

Build time: %s  Version: %s  Commit: %s
Github: https://github.com/wwwangzilin/LotsACG-Standalone
Kawaii is All You Need! ᕕ(◠ڼ◠)ᕗ

`

func Run() {
	commit := version.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	fmt.Printf(banner, version.BuildTime, version.Version, commit)
	ouid.MarshalJSON = json.Marshal
	ouid.UnmarshalJSON = json.Unmarshal

	cfg := runtimecfg.Get()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg, stop); err != nil {
		log.Fatal(err)
	}
}
