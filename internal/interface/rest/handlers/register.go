package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
)

func Register(router fiber.Router, serv *service.Service, cfg runtimecfg.RestConfig) {
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

	tgbotGroup := router.Group("/bot")
	tgbotGroup.Get("/status", HandleBotStatus)
	tgbotGroup.Get("/send_artwork_info", HandleSendArtworkInfoByTelegramBot)
	tgbotGroup.Post("/post_artwork", HandlePostArtworkToChannel)
	tgbotGroup.Post("/send_artwork_info", HandleSendArtworkInfoByTelegramBot)
}
