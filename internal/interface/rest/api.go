package rest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/handlers"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"

	"github.com/samber/oops"
)

type RestApp struct {
	fiberApp *fiber.App
	cfg      runtimecfg.RestConfig
	tgbot    common.TelegramBot
}

type RestAppOption func(app *RestApp) error

func WithTelegramBot(bot common.TelegramBot) RestAppOption {
	return func(app *RestApp) error {
		if bot == nil {
			return oops.New("telegram bot is nil")
		}
		app.tgbot = bot
		return nil
	}
}

func New(ctx context.Context, serv *service.Service, cfg runtimecfg.RestConfig, opts ...RestAppOption) (*RestApp, error) {
	errHandler := func(c fiber.Ctx, err error) error {
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		var e *common.Error
		if errors.As(err, &e) {
			if e.Status == fiber.StatusInternalServerError {
				log.Error("internal server error", "err", err, "url", c.OriginalURL())
			}
			return c.Status(e.Status).JSON(e.Response())
		}
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return c.Status(fe.Code).JSON(common.NewError(fe.Code, fe.Message).Response())
		}
		if errors.Is(err, errs.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(common.NewError(fiber.StatusNotFound, "resource not found").Response())
		}
		log.Error("internal server error", "err", err, "url", c.OriginalURL())
		code := fiber.StatusInternalServerError
		return c.Status(code).JSON(common.NewError(code, "internal server error").Response())
	}

	app := fiber.New(fiber.Config{
		JSONEncoder:     json.Marshal,
		JSONDecoder:     json.Unmarshal,
		ErrorHandler:    errHandler,
		StructValidator: NewStructValidator(),
		TrustProxy:      true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			LinkLocal: true,
			Private:   true,
			Loopback:  true,
		},
		ProxyHeader:        fiber.HeaderXForwardedFor,
		EnableIPValidation: true,
	})

	app.State().Set(common.StateKeyService, serv)
	app.State().Set(common.StateKeyConfig, cfg)

	if cfg.Limit.Enable {
		app.Use(limiter.New(limiter.Config{
			Expiration: time.Duration(cfg.Limit.Expiration) * time.Second,
			Max:        cfg.Limit.Max,
		}))
	}
	app.Use(cors.New())
	app.Use(etag.New())
	app.Use(compress.New())
	app.Use(recoverer.New())

	loggerCfg := logger.ConfigDefault
	loggerCfg.Format = "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${queryParams} | ${error}\n"
	app.Use(logger.New(loggerCfg))

	v1group := app.Group("/api/v1")
	handlers.Register(v1group, serv, cfg)

	// 内置 Web 前端 (ManyACG/web 构建产物), 与 API 同源托管
	if cfg.WebDir != "" {
		indexPath := filepath.Join(cfg.WebDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			// 与 nuxt.config 中 routeRules 一致的重定向
			app.Get("/setu", func(c fiber.Ctx) error {
				return c.Redirect().To("/api/v1/artwork/random/preview")
			})
			app.Get("/sese", func(c fiber.Ctx) error {
				return c.Redirect().To("/api/v1/picture/random")
			})
			app.Get("/atom.xml", func(c fiber.Ctx) error {
				return c.Redirect().To("/api/v1/atom")
			})
			// 静态文件 (未命中的路径会继续向下匹配)
			app.Use(static.New(cfg.WebDir))
			// SPA 兜底: 非 /api/ 的 GET 请求返回 index.html, 交给前端路由
			app.Get("*", func(c fiber.Ctx) error {
				if strings.HasPrefix(c.Path(), "/api/") {
					return c.Next()
				}
				return c.SendFile(indexPath)
			})
		}
	}

	restApp := &RestApp{
		fiberApp: app,
		cfg:      cfg,
	}

	for _, opt := range opts {
		if err := opt(restApp); err != nil {
			return nil, oops.Wrapf(err, "applying option")
		}
	}
	if restApp.tgbot != nil {
		app.State().Set(common.StateKeyTelegramBot, restApp.tgbot)
	}
	return restApp, nil
}

func (r *RestApp) Run(stopCtx context.Context) error {
	err := r.fiberApp.Listen(r.cfg.Addr, fiber.ListenConfig{
		GracefulContext: stopCtx,
	})
	if err != nil {
		// 端口冲突/占用时给出可操作的排查提示, 而不是裸抛底层 bind 错误
		if isAddrInUse(err) {
			return fmt.Errorf("%w (端口 %s 已被占用: 请修改 config.toml 中 [rest] addr, 或用 netstat -ano | findstr %s 查找占用进程并结束, Windows 也可用 taskkill /PID <pid> /F)", err, r.cfg.Addr, addrPort(r.cfg.Addr))
		}
		return err
	}
	return nil
}

// isAddrInUse 判断错误是否为端口占用 (跨平台: Windows / Linux / macOS)。
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address") ||
		strings.Contains(msg, "bind: address already in use")
}

// addrPort 从 ":8080" / "127.0.0.1:8080" 提取端口号, 用于排查提示。
func addrPort(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return addr
}
