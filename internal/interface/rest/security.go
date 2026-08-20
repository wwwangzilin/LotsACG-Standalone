package rest

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// 敏感写操作路径前缀: 这些端点涉及更新/启停/删除, 未鉴权时禁止。
var sensitivePrefixes = []string{
	"/update", "/xppusher", "/kmua", "/backup", "/admin",
}

// authMiddleware 返回 API token 鉴权中间件。
// - 未配置 api_token: 放行 (保持向后兼容, 但 0.0.0.0 监听时会有启动告警)
// - 已配置: 写操作 (POST/PUT/PATCH/DELETE) 必须携带合法 token; 敏感路径强制要求; 只读 GET 放行
func authMiddleware(cfg runtimecfg.RestConfig) fiber.Handler {
	token := strings.TrimSpace(cfg.APIToken)
	requireAuth := token != ""
	return func(c fiber.Ctx) error {
		if !requireAuth {
			return c.Next()
		}
		method := c.Method()
		path := c.Path()

		// 只读 GET/HEAD 且非敏感路径: 放行 (公开浏览)
		if (method == fiber.MethodGet || method == fiber.MethodHead) && !isSensitive(path) {
			return c.Next()
		}

		// 写操作 / 敏感路径: 校验 token
		provided := c.Get("X-API-Token")
		if provided == "" {
			auth := c.Get(fiber.HeaderAuthorization)
			if strings.HasPrefix(auth, "Bearer ") {
				provided = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if provided == "" || provided != token {
			// 审计: 记录未授权访问
			log.Warn("api auth failed",
				"ip", c.IP(), "method", method, "path", path,
				"time", time.Now().Format(time.RFC3339))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: 需要 Authorization: Bearer <token> 或 X-API-Token",
			})
		}
		return c.Next()
	}
}

// isSensitive 判断路径是否为敏感管理路径。
func isSensitive(path string) bool {
	for _, p := range sensitivePrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// auditSensitive 审计中间件: 记录敏感操作 (写方法 + 管理路径)。
func auditSensitive() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		path := c.Path()
		// 敏感: 写操作或管理路径
		if method != fiber.MethodGet && method != fiber.MethodHead || isSensitive(path) {
			log.Info("api sensitive operation",
				"ip", c.IP(), "method", method, "path", path,
				"time", time.Now().Format(time.RFC3339))
		}
		return c.Next()
	}
}
