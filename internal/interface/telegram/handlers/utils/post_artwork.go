package utils

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/unvgo/ouid"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/command"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/query"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/pkg/mediatool"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
)

func doPostAndCreateArtwork(
	ctx context.Context,
	bot *telego.Bot,
	serv *service.Service,
	meta *metautil.MetaData,
	artwork *entity.CachedArtworkData,
	fromChatID telego.ChatID,
	toChatID telego.ChatID,
	messageID int,
) error {
	awInDb, err := serv.GetArtworkByURL(ctx, artwork.SourceURL)
	if err == nil {
		return oops.Errorf("artwork already exists in db: %s", awInDb.SourceURL)
	}
	if serv.CheckDeletedByURL(ctx, artwork.SourceURL) {
		return oops.Errorf("artwork is marked as deleted: %s", artwork.SourceURL)
	}
	showProgress := (fromChatID.ID != 0 || fromChatID.Username != "")
	useEdit := (messageID != 0)

	editReplyMarkupText := func(text string) {
		if !showProgress || !useEdit {
			return
		}
		_, err := bot.EditMessageReplyMarkup(ctx, telegoutil.EditMessageReplyMarkup(
			fromChatID,
			messageID,
			telegoutil.InlineKeyboard([]telego.InlineKeyboardButton{
				telegoutil.InlineKeyboardButton(text).WithCallbackData("noop"),
			}),
		))
		if err != nil {
			log.Warn("failed to edit reply markup", "err", err)
		}
	}
	// replyWaitMsg 回复 messageID 的消息
	replyWaitMsg := func(text string) {
		if !showProgress {
			return
		}
		go func() {
			if useEdit {
				_, err := bot.SendMessage(ctx, telegoutil.Message(fromChatID, text).WithReplyParameters(&telego.ReplyParameters{
					MessageID: messageID,
				}).WithParseMode(telego.ModeHTML))
				if err != nil {
					log.Warn("failed to send reply wait message", "err", err)
				}
				return
			}
			_, err := bot.SendMessage(ctx, telegoutil.Message(fromChatID, text).WithParseMode(telego.ModeHTML))
			if err != nil {
				log.Warn("failed to send reply wait message", "err", err)
			}
		}()
	}

	editReplyMarkupText("正在存储资源...")

	for i, pic := range artwork.Pictures {
		// 下载并存储图片, 同时计算 phash, thumbhash, width, height
		err = func() error {
			// 若缓存的 URL 使用的代理与当前配置不一致, 立即刷新避免下载失败重试浪费时间
			if cachedURLProxyMismatch(artwork.SourceURL, pic.Original) {
				log.Warn("cached picture URL uses different proxy, refreshing")
				if fresh, err := serv.FetchArtworkInfo(ctx, artwork.SourceURL); err == nil && fresh != nil {
					if i < len(fresh.Pictures) {
						pic.Original = fresh.Pictures[i].Original
						pic.Thumbnail = fresh.Pictures[i].Thumbnail
					}
				}
			}
			// 按源生成候选图源: twitter 走 twimg 反代; 其他走 pixiv 代理链 (代理优先 -> 官方兜底)
			candidates := BuildImageCandidates(artwork.SourceURL, pic.Original)
			var cachedFile *osutil.File
			var dlErr error
			for ci, candidate := range candidates {
				for retry := 0; retry < 3; retry++ {
					if retry > 0 {
						time.Sleep(time.Duration(retry) * time.Second)
					}
					cachedFile, dlErr = httpclient.DownloadWithCache(ctx, candidate, nil)
					if dlErr == nil {
						break
					}
					log.Warnf("download picture %d via %s attempt %d failed: %v", i, candidate, retry+1, dlErr)
				}
				if dlErr == nil {
					break
				}
				if ci < len(candidates)-1 {
					log.Warnf("picture %d all attempts failed via %s, trying next source", i, candidate)
				}
			}
			if dlErr != nil {
				// 所有图源都失败, 尝试重新拉取作品信息刷新 URL 后再试一次
				log.Warn("refreshing cached artwork due to download failure")
				if fresh, err := serv.FetchArtworkInfo(ctx, artwork.SourceURL); err == nil && fresh != nil {
					if i < len(fresh.Pictures) {
						pic.Original = fresh.Pictures[i].Original
						pic.Thumbnail = fresh.Pictures[i].Thumbnail
						candidates = BuildImageCandidates(artwork.SourceURL, pic.Original)
						for _, candidate := range candidates {
							cachedFile, dlErr = httpclient.DownloadWithCache(ctx, candidate, nil)
							if dlErr == nil {
								break
							}
							log.Warnf("download picture %d (refreshed) via %s failed: %v", i, candidate, dlErr)
						}
					}
				}
			}
			if dlErr != nil {
				return oops.Wrapf(dlErr, "failed to download picture %d", i)
			}
			defer cachedFile.Close()
			img, _, err := image.Decode(cachedFile)
			if err != nil {
				return oops.Wrapf(err, "failed to decode picture %d", i)
			}
			if pic.Phash == "" {
				phash, err := mediatool.GetImagePhash(img)
				if err != nil {
					return oops.Wrapf(err, "failed to get phash of picture %d", i)
				}
				pic.Phash = phash
			}
			if pic.Orb == "" {
				orb, err := mediatool.GetImageORBFeatures(img)
				if err != nil {
					return oops.Wrapf(err, "failed to get orb features of picture %d", i)
				}
				pic.Orb = orb
			}
			if pic.Width == 0 || pic.Height == 0 {
				w, h, err := mediatool.GetImgSize(img)
				if err != nil {
					return oops.Wrapf(err, "failed to get size of picture %d", i)
				}
				pic.Width = uint(w)
				pic.Height = uint(h)
			}
			if pic.ThumbHash == "" {
				thumbHash, err := mediatool.GetImageThumbHash(img)
				if err != nil {
					return oops.Wrapf(err, "failed to get thumb hash of picture %d", i)
				}
				pic.ThumbHash = thumbHash
			}
			var ext string
			ext, err = strutil.GetFileExtFromURL(pic.Original)
			if err != nil {
				mtype, err := mimetype.DetectFile(cachedFile.Name())
				if err != nil {
					return oops.Wrapf(err, "failed to detect mime type for picture %d", i)
				}
				ext = mtype.Extension()
			}
			filename := fmt.Sprintf("%s%s", strutil.MD5Hash(pic.Original), ext)
			info, err := serv.StorageSaveAllSize(ctx, cachedFile.Name(), fmt.Sprintf("/%s/%s", artwork.SourceType, artwork.Artist.UID), filename)
			if err != nil {
				return oops.Wrapf(err, "failed to save picture %d", i)
			}
			if info != nil {
				artwork.Pictures[i].StorageInfo = *info
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	// 处理 ugoira 的 original
	for _, ugoira := range artwork.UgoiraMetas {
		err := func() error {
			origZip := ugoira.MetaData.OriginalZip
			file, err := httpclient.DownloadWithCache(ctx, origZip, nil)
			if err != nil {
				return oops.Wrapf(err, "failed to download ugoira original zip")
			}
			defer file.Close()
			filename := fmt.Sprintf("%s.zip", strutil.MD5Hash(origZip))
			info, err := serv.StorageSaveOriginal(ctx, file, fmt.Sprintf("/%s/%s/ugoira", artwork.SourceType, artwork.Artist.UID), filename)
			if err != nil {
				return oops.Wrapf(err, "failed to save ugoira original zip")
			}
			ugoira.OriginalStorage = *info
			return nil
		}()
		if err != nil {
			return err
		}
	}
	for _, video := range artwork.Videos {
		// 下载并存储视频
		err := func() error {
			file, err := httpclient.DownloadWithCache(ctx, video.URL, nil)
			if err != nil {
				return oops.Wrapf(err, "failed to download video")
			}
			defer file.Close()
			ext, err := strutil.GetFileExtFromURL(video.URL)
			if err != nil {
				mtype, err := mimetype.DetectFile(file.Name())
				if err != nil {
					return oops.Wrapf(err, "failed to detect mime type for video")
				}
				ext = mtype.Extension()
			}
			filename := fmt.Sprintf("%s%s", strutil.MD5Hash(video.URL), ext)
			info, err := serv.StorageSaveOriginal(ctx, file, fmt.Sprintf("/%s/%s/video", artwork.SourceType, artwork.Artist.UID), filename)
			if err != nil {
				return oops.Wrapf(err, "failed to save video")
			}
			if info != nil {
				video.OriginalStorage = *info
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	if serv.ShouldTagNewArtwork() || serv.ShouldAIAutoTag() {
		editReplyMarkupText("正在推理作品标签...")
		if err := serv.AutoTagCachedArtwork(ctx, artwork); err != nil {
			log.Warn("auto tagging failed", "url", artwork.SourceURL, "err", err)
		}
	}

	editReplyMarkupText("正在发布到频道...")

	targetChatID := toChatID
	if targetChatID.ID == 0 && targetChatID.Username == "" {
		targetChatID = meta.ResolvePostChatID(artwork)
	}
	if targetChatID.ID == 0 && targetChatID.Username == "" {
		targetChatID = toChatID
	}
	results, err := SendArtworkMediaGroup(ctx, bot, serv, meta, targetChatID, artwork)
	if err != nil {
		return oops.Wrapf(err, "failed to send artwork media group")
	}
	if len(results) == 0 {
		return oops.New("no messages sent")
	}
	// 更新 cached artwork 的 TelegramInfo
	// 这里不用 UpdateCachedArtworkFileID , 因为还需要更新 message 信息
	for _, msg := range results {
		tginfo := shared.TelegramInfo{}
		tginfo.SetMessage(targetChatID.ID, msg.Message.MessageID, msg.Message.MediaGroupID)
		switch msg.Type {
		case MediaResultTypePhoto:
			tginfo.SetFileID(meta.BotID(), shared.TelegramMediaTypePhoto, msg.FileID)
			artwork.Pictures[msg.Index].TelegramInfo = tginfo
		case MediaResultTypeUgoira:
			tginfo.SetFileID(meta.BotID(), shared.TelegramMediaTypeVideo, msg.FileID)
			artwork.UgoiraMetas[msg.Index].TelegramInfo = tginfo
		case MediaResultTypeVideo:
			tginfo.SetFileID(meta.BotID(), shared.TelegramMediaTypeVideo, msg.FileID)
			artwork.Videos[msg.Index].TelegramInfo = tginfo
		default:
			log.Error("message has unknown media result type", "message_id", msg.Message.MessageID, "type", msg.Type)
		}
	}
	if err := serv.UpdateCachedArtwork(ctx, artwork); err != nil {
		return oops.Wrapf(err, "failed to update cached artwork after sending")
	}
	// 创建 artwork
	awId, err := ouid.FromObjectIDHex(artwork.ID)
	if err != nil {
		awId = ouid.New()
	}
	ent, err := serv.CreateArtwork(ctx, &command.ArtworkCreation{
		ID:          awId,
		Title:       artwork.Title,
		Description: artwork.Description,
		R18:         artwork.R18,
		SourceType:  artwork.SourceType,
		Artist: command.ArtworkArtistCreation{
			Name:     artwork.Artist.Name,
			UID:      artwork.Artist.UID,
			Username: artwork.Artist.Username,
		},
		SourceURL: artwork.SourceURL,
		Tags:      artwork.Tags,
		UgoiraMetas: func() []*command.ArtworkUgoiraCreation {
			ugos := make([]*command.ArtworkUgoiraCreation, len(artwork.UgoiraMetas))
			for i, ugoira := range artwork.UgoiraMetas {
				ugos[i] = &command.ArtworkUgoiraCreation{
					Index:           ugoira.OrderIndex,
					Data:            ugoira.MetaData,
					OriginalStorage: ugoira.OriginalStorage,
					TelegramInfo:    ugoira.TelegramInfo,
				}
			}
			return ugos
		}(),
		Pictures: func() []command.ArtworkPictureCreation {
			pics := make([]command.ArtworkPictureCreation, len(artwork.Pictures))
			for i, pic := range artwork.Pictures {
				pics[i] = command.ArtworkPictureCreation{
					Index:        pic.OrderIndex,
					Thumbnail:    pic.Thumbnail,
					Original:     pic.Original,
					Width:        pic.Width,
					Height:       pic.Height,
					Phash:        pic.Phash,
					Orb:          pic.Orb,
					ThumbHash:    pic.ThumbHash,
					TelegramInfo: pic.TelegramInfo,
					StorageInfo:  pic.StorageInfo,
				}
			}
			return pics
		}(),
		Videos: func() []command.ArtworkVideoCreation {
			videos := make([]command.ArtworkVideoCreation, len(artwork.Videos))
			for i, video := range artwork.Videos {
				videos[i] = command.ArtworkVideoCreation{
					Index:           video.OrderIndex,
					URL:             video.URL,
					Width:           video.Width,
					Height:          video.Height,
					DurationMs:      video.Duration,
					Poster:          video.Poster,
					MimeType:        video.MimeType,
					TelegramInfo:    video.TelegramInfo,
					OriginalStorage: video.OriginalStorage,
				}
			}
			return videos
		}(),
	})
	if err != nil {
		return oops.Wrapf(err, "failed to create artwork in db")
	}
	log.Info("created artwork", "id", ent.ID, "url", ent.SourceURL, "title", ent.Title, "pics", len(ent.Pictures))

	editReplyMarkupText("已发布到频道, 正在检测重复图片...")
	if !service.GetDupCheckEnabled(ctx) {
		log.Info("duplicate check is disabled, skipping", "url", artwork.SourceURL)
		return nil
	}
	newEnt, err := serv.GetArtworkByURL(ctx, artwork.SourceURL)
	if err != nil {
		return oops.Wrapf(err, "failed to get artwork by url for duplicate picture check")
	}
	for i, pic := range newEnt.Pictures {
		var similars []*entity.Picture
		if pic.Phash != "" {
			phashSims, err := serv.QueryPicturesByPhash(ctx, query.PicturesPhash{Input: pic.Phash, Distance: 10, Limit: 20})
			if err != nil {
				log.Error("failed to query pictures by phash", "phash", pic.Phash, "err", err)
				editReplyMarkupText(fmt.Sprintf("检测第%d张图片重复失败, 作品已发布", i+1))
				continue
			}
			similars = append(similars, phashSims...)
		}
		if pic.Orb != "" {
			orbCfg := runtimecfg.Get().Search
			orbSims, err := serv.QueryPicturesByORB(ctx, query.PicturesORB{Input: pic.Orb, MinMatches: orbCfg.OrbMinMatches, MinScore: orbCfg.OrbMinScore, Limit: 20})
			if err != nil {
				log.Error("failed to query pictures by orb", "orb", pic.Orb, "err", err)
			} else {
				similars = append(similars, orbSims...)
			}
		}
		if len(similars) == 0 {
			continue
		}
		seen := make(map[ouid.OUID]struct{}, len(similars))
		sims := make([]*entity.Picture, 0, len(similars))
		for _, sim := range similars {
			if sim.ArtworkID == ent.ID {
				continue
			}
			if _, ok := seen[sim.ID]; ok {
				continue
			}
			seen[sim.ID] = struct{}{}
			sims = append(sims, sim)
		}
		if len(sims) == 0 {
			continue
		}
		log.Info("found similar pictures", "artwork_id", ent.ID, "picture_id", pic.ID, "count", len(sims), "similars", func() []string {
			ids := make([]string, len(sims))
			for i, s := range sims {
				ids[i] = s.ID.String()
			}
			return ids
		}())
		var text strings.Builder
		text.WriteString(fmt.Sprintf("检测到 %d 张与作品 <a href='%s'>%s 第 %d 张图片</a>相似的图片", len(sims), func() string {
			if meta.ChannelAvailable() { // 非 telegram handler context 下 meta 可能为 nil
				if msgId := pic.TelegramInfo.Data().MessageID(meta.ChannelChatID().ID); msgId != 0 {
					return meta.ChannelMessageURL(msgId)
				}
			}
			return ent.SourceURL
		}(), EscapeHTML(ent.Title), pic.OrderIndex))
		for j, sim := range sims {
			text.WriteString(fmt.Sprintf("\n\n%d - <a href='%s'>%s_%d</a>", j+1, func() string {
				if meta.ChannelAvailable() {
					if msgId := sim.TelegramInfo.Data().MessageID(meta.ChannelChatID().ID); msgId != 0 {
						return meta.ChannelMessageURL(msgId)
					}
				}
				return sim.Artwork.SourceURL
			}(), EscapeHTML(sim.Artwork.Title), sim.OrderIndex))
		}
		replyWaitMsg(text.String())
	}
	log.Debug("similar picture detection completed", "artwork_id", ent.ID)
	// recaption the posted message
	ent, err = serv.GetArtworkByURL(ctx, artwork.SourceURL)
	if err != nil {
		return oops.Wrapf(err, "failed to get artwork by url for recaption")
	}
	caption := ArtworkHTMLCaption(ent)
	firstMedia := ent.FirstMedia()
	msgID := 0
	if firstMedia != nil {
		msgID = firstMedia.GetTelegramInfo().MessageID(targetChatID.ID)
	}
	if msgID == 0 {
		log.Warn("no channel message id found for recaption, skip", "artwork", ent.SourceURL, "chat_id", targetChatID.ID)
		return nil
	}
	_, err = bot.EditMessageCaption(ctx, telegoutil.
		EditMessageCaption(targetChatID,
			msgID,
			caption).
		WithParseMode(telego.ModeHTML))
	if err != nil {
		log.Warn("failed to recaption posted artwork message", "err", err)
	}
	return nil
}

var (
	postArtworkSemaphore chan struct{}
)

func init() {
	const maxConcurrentPosts = 3
	postArtworkSemaphore = make(chan struct{}, maxConcurrentPosts)
}

func PostAndCreateArtwork(
	ctx context.Context,
	bot *telego.Bot,
	serv *service.Service,
	meta *metautil.MetaData,
	artwork *entity.CachedArtworkData,
	fromChatID telego.ChatID,
	toChatID telego.ChatID,
	messageID int,
) error {
	select {
	case postArtworkSemaphore <- struct{}{}:
		defer func() { <-postArtworkSemaphore }()
		return doPostAndCreateArtwork(ctx, bot, serv, meta, artwork, fromChatID, toChatID, messageID)
	case <-ctx.Done():
		return ctx.Err()
	}
}
