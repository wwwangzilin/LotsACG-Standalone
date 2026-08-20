package handlers

import (
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/filter"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
)

type HandlerManager struct {
	*metautil.MetaData
	*service.Service
}

func New(meta *metautil.MetaData, serv *service.Service) *HandlerManager {
	return &HandlerManager{
		MetaData: meta,
		Service:  serv,
	}
}

func (m HandlerManager) Register(hg *telegohandler.HandlerGroup) {
	hg.Handle(func(ctx *telegohandler.Context, update telego.Update) error {
		servCtx := service.WithContext(ctx, m.Service)
		metaCtx := metautil.WithContext(servCtx, m.MetaData)
		ctx = ctx.WithContext(metaCtx)
		if !m.checkUserAllowed(update) {
			if update.Message != nil {
				utils.ReplyMessage(ctx, *update.Message, "仅允许配置中登记的用户使用该功能")
			}
			return nil
		}
		return ctx.Next(update)
	})
	mg := hg.Group(telegohandler.AnyMessage(), filter.CommandToMe)
	mg.HandleMessage(Start, telegohandler.CommandEqual("start"))
	mg.HandleMessage(GetArtworkFiles, telegohandler.Or(telegohandler.CommandEqual("file"), telegohandler.CommandEqual("files")))
	mg.HandleMessage(RandomPicture, telegohandler.Or(telegohandler.CommandEqual("setu"), telegohandler.CommandEqual("random")))
	mg.HandleMessage(Help, telegohandler.CommandEqual("help"))
	mg.HandleMessage(SearchPicture, telegohandler.CommandEqual("search"))
	mg.HandleMessage(GetArtworkInfoCommand, telegohandler.CommandEqual("info"))
	mg.HandleMessage(HybridSearchArtworks, telegohandler.CommandEqual("hybrid"))
	mg.HandleMessage(SearchSimilarArtworks, telegohandler.CommandEqual("similar"))
	mg.HandleMessage(Status, telegohandler.CommandEqual("status"))
	mg.HandleMessage(R18ModeCmd, telegohandler.CommandEqual("r18mode"))
	mg.HandleMessage(GroupSearch, telegohandler.CommandEqual("groupsearch"))
	mg.HandleMessage(DownloadZip, telegohandler.CommandEqual("downloadzip"))
	mg.HandleMessage(TaggingPicture, telegohandler.CommandEqual("tagging"))
	mg.HandleMessage(FollowArtist, telegohandler.CommandEqual("follow"))
	mg.HandleMessage(UnfollowArtist, telegohandler.CommandEqual("unfollow"))
	mg.HandleMessage(FollowList, telegohandler.CommandEqual("followlist"))
	mg.HandleMessage(SubscribeTag, telegohandler.CommandEqual("sub"))
	mg.HandleMessage(UnsubscribeTag, telegohandler.CommandEqual("unsub"))
	mg.HandleMessage(TagSubList, telegohandler.CommandEqual("sublist"))

	// Admin commands
	mg.HandleMessage(SetAdmin, telegohandler.Or(telegohandler.CommandEqual("addadmin"), telegohandler.CommandEqual("deladmin")))
	mg.HandleMessage(DeleteArtwork, telegohandler.Or(telegohandler.CommandEqual("delete"), telegohandler.CommandEqual("del")))
	mg.HandleMessage(ToggleArtworkR18, telegohandler.CommandEqual("r18"))
	mg.HandleMessage(SetArtworkTags, telegohandler.Or(telegohandler.CommandEqual("tags"), telegohandler.CommandEqual("addtags"), telegohandler.CommandEqual("deltags")))
	mg.HandleMessage(EditArtworkTitle, telegohandler.CommandEqual("title"))
	mg.HandleMessage(PostArtworkCommand, telegohandler.CommandEqual("post"))
	mg.HandleMessage(RefreshArtwork, telegohandler.CommandEqual("refresh"))
	mg.HandleMessage(AddTagAlias, telegohandler.CommandEqual("tagalias"))
	mg.HandleMessage(DumpArtworkInfo, telegohandler.CommandEqual("dump"))
	mg.HandleMessage(ReCaptionArtwork, telegohandler.CommandEqual("recaption"))
	mg.HandleMessage(AutoTaggingArtwork, telegohandler.CommandEqual("autotag"))
	mg.HandleMessage(ReindexArtworks, telegohandler.CommandEqual("reindex"))
	mg.HandleMessage(ToggleDupCheck, telegohandler.CommandEqual("dupcheck"))
	mg.HandleMessage(CancelPost, telegohandler.CommandEqual("cancel"))
	mg.HandleMessage(CancelAndDeletePost, telegohandler.CommandEqual("cd"))
	mg.HandleMessage(RedescribeArtwork, telegohandler.CommandEqual("redescribe"))
	mg.HandleMessage(UpdateCmd, telegohandler.CommandEqual("update"))
	mg.HandleMessage(RollbackCmd, telegohandler.CommandEqual("rollback"))
	mg.HandleMessage(XPPusher, telegohandler.CommandEqual("xppusher"))
	mg.HandleMessage(KMua, telegohandler.CommandEqual("kmua"))
	mg.HandleMessage(Channel, telegohandler.CommandEqual("channel"))

	hg.HandleCallbackQuery(PostArtworkCallbackQuery, telegohandler.CallbackDataContains("post_artwork"))
	hg.HandleCallbackQuery(SearchPictureCallbackQuery, telegohandler.CallbackDataPrefix("search_picture"))
	hg.HandleCallbackQuery(EditArtworkR18, telegohandler.CallbackDataPrefix("edit_artwork r18"))
	hg.HandleCallbackQuery(DeleteArtworkCallbackQuery, telegohandler.CallbackDataPrefix("delete_artwork"))
	hg.HandleCallbackQuery(SendtoArtworkCallbackQuery, telegohandler.CallbackDataPrefix("sendto"))
	hg.HandleCallbackQuery(UpdateConfirmCallback, telegohandler.CallbackDataEqual("update_confirm"))
	hg.HandleCallbackQuery(UpdateCancelCallback, telegohandler.CallbackDataEqual("update_cancel"))

	hg.HandleInlineQuery(InlineQuery)
	hg.Use(func(ctx *telegohandler.Context, update telego.Update) error {
		msg := update.Message
		if msg == nil {
			return ctx.Err()
		}
		if update.Message.ViaBot != nil && update.Message.ViaBot.Username == m.BotUsername() {
			return ctx.Err()
		}
		// 画师主页链接: 自动输出该画师全部作品的完整链接 (仅链接, 无说明)
		if artistURL := utils.FindArtistPageURLInMessage(m.Service, update.Message); artistURL != "" {
			return handleArtistPageURL(ctx, m, *update.Message, artistURL)
		}
		if url := utils.FindSourceURLInMessage(m.Service, update.Message); url != "" {
			ctx = ctx.WithValue("source_url", url)
			return ctx.Next(update)
		}
		return ctx.Err()
	})
	hg.HandleMessage(GetArtworkInfo)
}

// checkUserAllowed 白名单检查。仅当配置了 allowed_users (或 admins) 时启用:
// 未登记的用户(游客)只允许 /start /help /files(获取原图)。
// 普通非命令消息(闲聊/发图等)不拦截; callback / inline 查询一律按白名单处理。
func (m HandlerManager) checkUserAllowed(update telego.Update) bool {
	allowed := m.MetaData.AllowedUsers()
	if len(allowed) == 0 {
		return true
	}
	var userID int64
	var cmd string
	if msg := update.Message; msg != nil && msg.From != nil {
		userID = msg.From.ID
		c, _, _ := telegoutil.ParseCommand(msg.Text)
		cmd = strings.ToLower(c)
		if cmd == "" {
			// 非命令消息(如闲聊、发图)不拦截
			return true
		}
	} else if cq := update.CallbackQuery; cq != nil {
		userID = cq.From.ID
	} else {
		// 非用户消息(如 channel post / 无 From)不拦截
		return true
	}
	// 游客允许的命令
	switch cmd {
	case "start", "help", "file", "files":
		return true
	}
	for _, id := range allowed {
		if id == userID {
			return true
		}
	}
	return false
}
