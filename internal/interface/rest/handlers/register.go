package handlers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

func Register(router fiber.Router, serv *service.Service, cfg runtimecfg.RestConfig) {
	// 可观测性端点：健康检查 / 就绪探针 / Prometheus 指标 / 增强状态
	router.Get("/healthz", HandleHealthz)
	router.Get("/readyz", HandleReadyz)
	router.Get("/metrics", HandleMetrics)
	router.Get("/status/ex", HandleStatusEx)

	// 注入指标收集器：作品总数 + bot 队列深度
	RegisterMetricsCollector(MetricsCollector{
		Artworks: func() float64 {
			count, err := serv.CountArtworks(context.Background(), shared.R18TypeAll)
			if err != nil {
				return 0
			}
			return float64(count)
		},
	})

	router.Get("/atom", GenerateAtomFeed)
	router.Get("/myip", MyIP(cfg))

	artworkGroup := router.Group("/artwork")
	artworkGroup.Get("/random/preview", HandleRandomPreviewArtworks)
	artworkGroup.Get("/random", HandleRandomArtworks)
	artworkGroup.Post("/random", HandleRandomArtworks)
	artworkGroup.Get("/list", GetHandleListArtworks(serv, cfg))
	artworkGroup.Post("/list", GetHandleListArtworks(serv, cfg))
	artworkGroup.Get("/count", HandleCountArtwork)
	artworkGroup.Get("/fetch", HandleFetchArtwork)
	artworkGroup.Post("/fetch", HandleFetchArtwork)
	artworkGroup.Get("/:id", HandleGetArtworkByID)

	pictureGroup := router.Group("/picture")
	pictureGroup.Get("/file/:size/:id", HandleGetSizedPictureFileByID)
	pictureGroup.Get("/file/:id", HandleGetPictureFileByID)
	pictureGroup.Get("/random", HandleGetRandomPicture)

	artistGroup := router.Group("/artist")
	artistGroup.Get("/:id", HandleGetArtistByID)

	tagGroup := router.Group("/tag")
	tagGroup.Get("/random", HandleGetRandomTags)

	// 混合搜索: 文本 / 以图搜文 统一入口
	searchGroup := router.Group("/search")
	searchGroup.Get("/hybrid", HandleHybridSearch(serv))
	searchGroup.Post("/hybrid", HandleHybridSearch(serv))

	tgbotGroup := router.Group("/bot")
	tgbotGroup.Get("/status", HandleBotStatus)
	tgbotGroup.Get("/send_artwork_info", HandleSendArtworkInfoByTelegramBot)
	tgbotGroup.Post("/post_artwork", HandlePostArtworkToChannel)
	tgbotGroup.Post("/send_artwork_info", HandleSendArtworkInfoByTelegramBot)
}
