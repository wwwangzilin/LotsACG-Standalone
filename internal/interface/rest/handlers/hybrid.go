package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
)

// RequestHybridSearch 混合搜索请求: 支持 query (文本) 与 image_url (以图搜文)。
type RequestHybridSearch struct {
	Query    string `json:"query" query:"query" form:"query"`
	ImageURL string `json:"image_url" query:"image_url" form:"image_url"`
	Limit    int    `json:"limit" query:"limit" form:"limit"`
}

// HandleHybridSearch 统一混合搜索端点 (Web 与 Telegram 共用同一 service)。
// GET/POST /api/v1/search/hybrid?query=...&image_url=...
func HandleHybridSearch(serv *service.Service) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		req := new(RequestHybridSearch)
		if err := ctx.Bind().All(req); err != nil {
			return err
		}
		result, err := serv.HybridSearch(ctx.Context(), &service.HybridSearchRequest{
			Query:    req.Query,
			ImageURL: req.ImageURL,
			Limit:    req.Limit,
		})
		if err != nil {
			return common.NewError(fiber.StatusBadRequest, "hybrid search failed: "+err.Error())
		}
		if result == nil {
			return common.NewError(fiber.StatusBadRequest, "provide query or image_url")
		}
		return ctx.JSON(common.NewSuccess(result))
	}
}
