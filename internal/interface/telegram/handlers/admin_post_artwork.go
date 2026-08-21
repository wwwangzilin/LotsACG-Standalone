package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

func PostArtworkCallbackQuery(ctx *telegohandler.Context, query telego.CallbackQuery) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionForQuery(ctx, serv, query, shared.PermissionPostArtwork) {
		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "你没有发布作品的权限",
			ShowAlert:       true,
			CacheTime:       60,
		})
		return nil
	}
	queryDataSlice := strings.Split(query.Data, " ")
	reverseR18 := queryDataSlice[0] == "post_artwork_r18"
	dataID := queryDataSlice[1]
	sourceURL, err := kvstor.Get[string](ctx, dataID)
	if err != nil {
		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "获取回调数据失败",
			ShowAlert:       true,
			CacheTime:       60,
		})
		return nil
	}
	cachedArtwork, err := serv.GetOrFetchCachedArtwork(ctx, sourceURL)
	if err != nil {
		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "获取作品信息失败 " + err.Error(),
			ShowAlert:       true,
			CacheTime:       60,
		})
		return nil
	}
	if cachedArtwork.Status == shared.ArtworkStatusPosting {
		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "该作品正在发布中",
			ShowAlert:       true,
			CacheTime:       60,
		})
		return nil
	}

	if err := serv.UpdateCachedArtworkStatusByURL(ctx, sourceURL, shared.ArtworkStatusPosting); err != nil {
		log.Errorf("更新缓存作品状态失败: %s", err)
		return oops.Wrapf(err, "failed to update cached artwork status")
	}
	log.Info("posting artwork", "url", sourceURL)

	artwork := cachedArtwork.Artwork.Data()
	ctx.Bot().EditMessageCaption(ctx, &telego.EditMessageCaptionParams{
		ChatID:      telegoutil.ID(query.Message.GetChat().ID),
		MessageID:   query.Message.GetMessageID(),
		Caption:     fmt.Sprintf("正在发布: %s", artwork.SourceURL),
		ReplyMarkup: nil,
	})
	if err := serv.CancelDeletedByURL(ctx, sourceURL); err != nil {
		log.Errorf("取消删除记录失败: %s", err)
		ctx.Bot().EditMessageCaption(ctx, &telego.EditMessageCaptionParams{
			ChatID:    telegoutil.ID(query.Message.GetChat().ID),
			MessageID: query.Message.GetMessageID(),
			Caption:   "取消删除记录失败: " + err.Error(),
		})
		return nil
	}
	if reverseR18 {
		artwork.R18 = !artwork.R18
	}
	meta, err := requireMeta(ctx)
	if err != nil {
		return err
	}
	if meta.ChannelAvailable() {
		if err := utils.PostAndCreateArtwork(ctx, ctx.Bot(), serv, meta, artwork, query.Message.GetChat().ChatID(), meta.ChannelChatID(), query.Message.GetMessageID()); err != nil {
			log.Errorf("failed to post and create artwork: %s", err)
			ctx.Bot().EditMessageCaption(ctx, &telego.EditMessageCaptionParams{
				ChatID:    telegoutil.ID(query.Message.GetChat().ID),
				MessageID: query.Message.GetMessageID(),
				Caption:   "发布失败: " + err.Error() + "\n" + time.Now().Format("2006-01-02 15:04:05"),
			})
			if err := serv.UpdateCachedArtworkStatusByURL(ctx, sourceURL, shared.ArtworkStatusCached); err != nil {
				// log.Warnf("更新缓存作品状态失败: %s", err)
				log.Error("failed to update cached artwork status", "err", err)
			}
			return nil
		}
		awEnt, err := serv.GetArtworkByURL(ctx, sourceURL)
		if err != nil {
			return oops.Wrapf(err, "failed to get created artwork by url")
		}
		_, err = ctx.Bot().EditMessageCaption(ctx, &telego.EditMessageCaptionParams{
			ChatID:      telegoutil.ID(query.Message.GetChat().ID),
			MessageID:   query.Message.GetMessageID(),
			Caption:     fmt.Sprintf("发布成功: %s / %s\n%s", awEnt.Title, awEnt.GetSourceURL(), time.Now().Format("2006-01-02 15:04:05")),
			ReplyMarkup: telegoutil.InlineKeyboard(utils.GetPostedArtworkInlineKeyboardButton(awEnt, meta)),
		})
		return err
	}
	return nil
}

func PostArtworkCommand(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionPostArtwork) {
		return oops.Errorf("user %d has no permission to post artwork", message.From.ID)
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	if len(args) == 0 && message.ReplyToMessage == nil {
		utils.ReplyMessage(ctx, message, "请提供作品链接, 或回复一条消息")
		return nil
	}

	sourceURLs := make([]string, 0, len(args))
	if message.ReplyToMessage != nil {
		sourceURLs = append(sourceURLs, utils.FindSourceURLsInMessage(serv, message.ReplyToMessage)...)
		// 回复的消息中也可能包含画师主页链接
		if artistURL := utils.FindArtistPageURLInMessage(serv, message.ReplyToMessage); artistURL != "" {
			sourceURLs = append(sourceURLs, artistURL)
		}
	}
	for _, arg := range args {
		if sourceURL := serv.FindSourceURL(arg); sourceURL != "" {
			sourceURLs = append(sourceURLs, sourceURL)
			continue
		}
		// 画师主页链接: 加入待展开列表 (后续会替换为其全部作品链接)
		if artistURL := serv.FindArtistPageURL(arg); artistURL != "" {
			sourceURLs = append(sourceURLs, artistURL)
		}
	}
	if len(sourceURLs) == 0 {
		sourceURLs = append(sourceURLs, utils.FindSourceURLsInMessage(serv, &message)...)
		if artistURL := utils.FindArtistPageURLInMessage(serv, &message); artistURL != "" {
			sourceURLs = append(sourceURLs, artistURL)
		}
	}
	if len(sourceURLs) == 0 {
		utils.ReplyMessage(ctx, message, "不支持的链接")
		return nil
	}

	// 展开画师主页链接: 将作者主页替换为其全部作品链接
	expanded := make([]string, 0, len(sourceURLs)*2)
	artistURLs := make([]string, 0)
	for _, sourceURL := range sourceURLs {
		if serv.FindArtistPageURL(sourceURL) != "" {
			urls, err := serv.FetchArtistArtworks(ctx, sourceURL, 0)
			if err != nil {
				log.Warnf("post: failed to fetch artist artworks for %s: %v", sourceURL, err)
				utils.ReplyMessage(ctx, message, "获取画师作品失败: "+err.Error())
				return nil
			}
			if len(urls) == 0 {
				utils.ReplyMessage(ctx, message, "该画师暂无作品")
				return nil
			}
			log.Info("post: expand artist page", "url", sourceURL, "count", len(urls))
			expanded = append(expanded, urls...)
			artistURLs = append(artistURLs, sourceURL)
		} else {
			expanded = append(expanded, sourceURL)
		}
	}
	sourceURLs = expanded

	// 发布画师主页时自动订阅该画师: 之后新作品将自动补齐发布到主频道
	for _, artistURL := range artistURLs {
		if added, err := serv.FollowArtist(ctx, message.From.ID, artistURL); err != nil {
			log.Warnf("post: failed to auto subscribe artist %s: %v", artistURL, err)
		} else if added {
			log.Info("post: auto subscribed artist", "url", artistURL, "user", message.From.ID)
		}
	}

	seen := make(map[string]struct{}, len(sourceURLs))
	uniqueSourceURLs := make([]string, 0, len(sourceURLs))
	for _, sourceURL := range sourceURLs {
		if _, ok := seen[sourceURL]; ok {
			continue
		}
		seen[sourceURL] = struct{}{}
		uniqueSourceURLs = append(uniqueSourceURLs, sourceURL)
	}

	msg, err := utils.ReplyMessage(ctx, message, fmt.Sprintf("正在排队发布 %d 条作品...", len(uniqueSourceURLs)))
	if err != nil || msg == nil {
		msg = nil
	}

	meta, err := requireMeta(ctx)
	if err != nil {
		return err
	}
	if !meta.ChannelAvailable() {
		utils.ReplyMessage(ctx, message, "频道未配置, 无法发布")
		return nil
	}

	successCount := 0
	skipCount := 0
	failCount := 0
	results := make([]string, 0, len(uniqueSourceURLs))

	// 新建发布队列 (供 /cancel 取消、/cd 取消并删除)
	queue := &PostQueue{
		SourceURLs: uniqueSourceURLs,
		Status:     postQueueStatusPosting,
		CreatedAt:  time.Now(),
	}
	_ = savePostQueue(ctx, queue)

	// 进度跟踪: 已用时长 + 预计完成用时 (每 1 分钟更新一次), 并统计下载字节数
	startTime := time.Now()
	var bytesDown int64
	postCtx := httpclient.WithBytesCounter(ctx, &bytesDown)
	progress := &postProgress{total: len(uniqueSourceURLs), start: startTime}
	stopProgressCh := make(chan struct{})
	var progressWg sync.WaitGroup
	if msg != nil {
		progressWg.Add(1)
		go func() {
			defer progressWg.Done()
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-stopProgressCh:
					return
				case <-ticker.C:
					editPostProgress(ctx, msg, progress)
				}
			}
		}()
	}

	for idx, sourceURL := range uniqueSourceURLs {
		// 检查是否被 /cancel 或 /cd 取消
		if cur, err := loadPostQueue(postCtx); err == nil && cur.Status == postQueueStatusCancelled {
			results = append(results, fmt.Sprintf("队列已取消, 停止于 %d/%d: %s", idx+1, len(uniqueSourceURLs), sourceURL))
			break
		}
		progress.mu.Lock()
		progress.idx = idx + 1
		progress.url = sourceURL
		progress.mu.Unlock()
		editPostProgress(ctx, msg, progress)

		awEnt, _ := serv.GetArtworkByURL(postCtx, sourceURL)
		if awEnt != nil {
			skipCount++
			results = append(results, fmt.Sprintf("%d/%d 已存在: %s", idx+1, len(uniqueSourceURLs), sourceURL))
			continue
		}

		cachedArtwork, err := serv.GetOrFetchCachedArtwork(postCtx, sourceURL)
		if err != nil {
			log.Errorf("failed to get or fetch cached artwork: %s", err)
			failCount++
			results = append(results, fmt.Sprintf("%d/%d 获取作品信息失败: %s", idx+1, len(uniqueSourceURLs), sourceURL))
			continue
		}
		if cachedArtwork.Status != shared.ArtworkStatusCached {
			skipCount++
			results = append(results, fmt.Sprintf("%d/%d 已发布或正在发布中: %s", idx+1, len(uniqueSourceURLs), sourceURL))
			continue
		}
		artwork := cachedArtwork.Artwork.Data()
		if err := utils.PostAndCreateArtwork(postCtx, ctx.Bot(), serv, meta, artwork, message.GetChat().ChatID(), meta.ChannelChatID(), message.MessageID); err != nil {
			failCount++
			results = append(results, fmt.Sprintf("%d/%d 发布失败: %s\n  原因: %v", idx+1, len(uniqueSourceURLs), sourceURL, err))
			continue
		}
		createdArtwork, err := serv.GetArtworkByURL(postCtx, sourceURL)
		if err != nil {
			failCount++
			results = append(results, fmt.Sprintf("%d/%d 发布后查找作品失败: %s", idx+1, len(uniqueSourceURLs), sourceURL))
			continue
		}
		successCount++
		results = append(results, fmt.Sprintf("%d/%d 发布成功: %s / %s", idx+1, len(uniqueSourceURLs), createdArtwork.Title, createdArtwork.GetSourceURL()))
		// 记录已发布, 供 /cd 删除
		if cur, err := loadPostQueue(postCtx); err == nil {
			cur.Published = append(cur.Published, sourceURL)
			_ = savePostQueue(postCtx, cur)
		}
	}

	// 停止进度更新 goroutine
	close(stopProgressCh)
	progressWg.Wait()

	// 队列完成 (若未被取消)
	if cur, err := loadPostQueue(ctx); err == nil && cur.Status != postQueueStatusCancelled {
		cur.Status = postQueueStatusDone
		_ = savePostQueue(ctx, cur)
	}

	elapsed := time.Since(startTime)
	finalText := fmt.Sprintf("发布完成: 成功 %d, 跳过 %d, 失败 %d\n总用时: %s", successCount, skipCount, failCount, formatPostDuration(elapsed))
	if bytesDown > 0 {
		speedMBps := float64(bytesDown) / 1024 / 1024 / elapsed.Seconds()
		finalText += fmt.Sprintf("\n平均下载速度: %.2f MB/s (共 %.1f MB)", speedMBps, float64(bytesDown)/1024/1024)
	}
	if len(results) > 0 {
		finalText += "\n" + strings.Join(results, "\n")
	}
	if msg != nil {
		ctx.Bot().EditMessageText(ctx, telegoutil.EditMessageText(msg.Chat.ChatID(), msg.MessageID, finalText))
	} else {
		utils.ReplyMessage(ctx, message, finalText)
	}
	return nil
}

// postProgress 记录批量发布的进度状态, 供 1 分钟定时更新预计完成用时。
type postProgress struct {
	mu     sync.Mutex // 保护进度状态
	editMu sync.Mutex // 串行化对进度消息的编辑
	idx    int
	total  int
	url    string
	start  time.Time
}

// editPostProgress 编辑进度消息: 当前进度 + 已用时长 + 预计剩余/完成时间。
func editPostProgress(ctx *telegohandler.Context, msg *telego.Message, p *postProgress) {
	if msg == nil || p == nil {
		return
	}
	p.editMu.Lock()
	defer p.editMu.Unlock()
	p.mu.Lock()
	idx, total, url, start := p.idx, p.total, p.url, p.start
	p.mu.Unlock()
	elapsed := time.Since(start)
	text := fmt.Sprintf("正在发布 %d/%d", idx, total)
	if url != "" {
		text += ": " + url
	}
	text += "\n已用时长: " + formatPostDuration(elapsed)
	if idx > 0 && idx < total {
		perItem := elapsed / time.Duration(idx)
		remaining := perItem * time.Duration(total-idx)
		text += "\n预计剩余: 约 " + formatPostDuration(remaining)
		text += " (预计完成: " + time.Now().Add(remaining).Format("15:04:05") + ")"
	}
	_, _ = ctx.Bot().EditMessageText(ctx, telegoutil.EditMessageText(msg.Chat.ChatID(), msg.MessageID, text))
}

// formatPostDuration 将时长格式化为中文可读形式 (如 "3分20秒" / "1小时2分3秒")。
func formatPostDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d / time.Hour)
	m := int(d % time.Hour / time.Minute)
	s := int(d % time.Minute / time.Second)
	switch {
	case h > 0:
		return fmt.Sprintf("%d小时%d分%d秒", h, m, s)
	case m > 0:
		return fmt.Sprintf("%d分%d秒", m, s)
	default:
		return fmt.Sprintf("%d秒", s)
	}
}

// func ArtworkPreview(ctx *telegohandler.Context, query telego.CallbackQuery) error {
// 	serv := service.FromContext(ctx)
// 	if !utils.CheckPermissionForQuery(ctx, serv, query, shared.PermissionPostArtwork) {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "你没有发布作品的权限",
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}
// 	queryDataSlice := strings.Split(query.Data, " ")
// 	dataID := queryDataSlice[1]
// 	sourceURL, err := serv.GetStringDataByID(ctx, dataID)
// 	if err != nil {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "获取回调数据失败 " + err.Error(),
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}
// 	cachedArtworkEnt, err := serv.GetOrFetchCachedArtwork(ctx, sourceURL)
// 	if err != nil {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "获取作品信息失败 " + err.Error(),
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}
// 	if cachedArtworkEnt.Status != shared.ArtworkStatusCached {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "该作品已发布或正在发布中",
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}
// 	cachedArtwork := cachedArtworkEnt.Artwork.Data()
// 	var callbackMessage *telego.Message
// 	if query.Message.IsAccessible() {
// 		callbackMessage = query.Message.(*telego.Message)
// 	} else {
// 		log.Warnf("callback message is not accessible")
// 		return nil
// 	}
// 	meta := metautil.FromContext(ctx)
// 	postArtworkKeyboard := [][]telego.InlineKeyboardButton{
// 		{
// 			telegoutil.InlineKeyboardButton("发布").WithCallbackData("post_artwork " + dataID),
// 			telegoutil.InlineKeyboardButton("发布(反转R18)").WithCallbackData("post_artwork_r18 " + dataID),
// 		},
// 		{
// 			telegoutil.InlineKeyboardButton("查重").WithCallbackData("search_picture " + dataID),
// 			telegoutil.InlineKeyboardButton("预览发布").WithURL(meta.BotDeepLink("info", dataID)),
// 		},
// 	}

// 	currentPictureIndexStr := queryDataSlice[4]
// 	// 此处为当前图片在 cachedArtwork.Pictures 中的 Index 字段, 从0开始
// 	// 由于隐藏机制的存在, 呈递到界面的图片索引不一定连续
// 	currentPictureIndex, err := strconv.Atoi(currentPictureIndexStr)
// 	if err != nil {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "解析回调数据错误: " + err.Error(),
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}
// 	opera := queryDataSlice[2]
// 	if opera == "delete" {
// 		if err := serv.HideCachedArtworkPicture(ctx, cachedArtworkEnt, currentPictureIndex); err != nil {
// 			ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 				CallbackQueryID: query.ID,
// 				Text:            "删除失败: " + err.Error(),
// 				ShowAlert:       true,
// 				CacheTime:       60,
// 			})
// 			return nil
// 		}
// 		cachedArtworkEnt, err = serv.GetCachedArtworkByURL(ctx, sourceURL)
// 		if err != nil {
// 			ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 				CallbackQueryID: query.ID,
// 				Text:            "已删除该图片, 但获取更新信息失败: " + err.Error(),
// 				ShowAlert:       true,
// 				CacheTime:       60,
// 			})
// 			return nil
// 		}
// 		cachedArtwork = cachedArtworkEnt.Artwork.Data()
// 		go ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "删除成功, 稍后发布作品时将不包含该图片",
// 			CacheTime:       1,
// 		})

// 		previewKeyboard := []telego.InlineKeyboardButton{}

// 		if currentPictureIndex+1 >= len(cachedArtwork.GetViewablePictures()) {
// 			// 如果删除的是最后一张图片, 则显示前一张
// 			if currentPictureIndex > 0 {
// 				currentPictureIndex -= 1
// 				currentPictureIndexStr = strconv.Itoa(currentPictureIndex)
// 			}
// 		}

// 		if len(cachedArtwork.GetViewablePictures()) > 1 {

// 			deleteButton := telegoutil.InlineKeyboardButton(fmt.Sprintf("删除这张(%d)", currentPictureIndex+1)).WithCallbackData("awpv " + dataID + " delete " + currentPictureIndexStr + " " + currentPictureIndexStr)
// 			if currentPictureIndex == 0 {
// 				previewKeyboard = append(previewKeyboard,
// 					deleteButton,
// 					telegoutil.InlineKeyboardButton("下一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", currentPictureIndex+1, currentPictureIndex)),
// 				)
// 			} else if currentPictureIndex == len(cachedArtwork.Pictures)-1 {
// 				previewKeyboard = append(previewKeyboard,
// 					telegoutil.InlineKeyboardButton("上一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", currentPictureIndex-1, currentPictureIndex)),
// 					deleteButton,
// 				)
// 			} else {
// 				previewKeyboard = append(previewKeyboard,
// 					telegoutil.InlineKeyboardButton("上一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", currentPictureIndex-1, currentPictureIndex)),
// 					deleteButton,
// 					telegoutil.InlineKeyboardButton("下一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", currentPictureIndex+1, currentPictureIndex)),
// 				)
// 			}
// 		}
// 		inputFile, err := utils.GetPicturePhotoInputFile(ctx, serv, cachedArtwork.Pictures[currentPictureIndex])
// 		if err != nil {
// 			log.Errorf("获取预览图片失败: %s", err)
// 			ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 				CallbackQueryID: query.ID,
// 				Text:            "获取预览图片失败: " + err.Error(),
// 				ShowAlert:       true,
// 				CacheTime:       60,
// 			})
// 			return nil
// 		}
// 		postArtworkKeyboard = append(postArtworkKeyboard, previewKeyboard)
// 		_, err = ctx.Bot().EditMessageMedia(ctx, &telego.EditMessageMediaParams{
// 			ChatID:      callbackMessage.Chat.ChatID(),
// 			MessageID:   callbackMessage.MessageID,
// 			ReplyMarkup: telegoutil.InlineKeyboard(postArtworkKeyboard...),
// 			Media: telegoutil.MediaPhoto(inputFile).
// 				WithCaption(utils.ArtworkHTMLCaption(meta, cachedArtwork) + fmt.Sprintf("\n<i>当前作品有 %d 张图片</i>", len(cachedArtwork.GetPictures()))).
// 				WithParseMode(telego.ModeHTML),
// 		})
// 		if err != nil {
// 			log.Errorf("编辑预览消息失败: %s", err)
// 		}
// 		return nil
// 	}

// 	pictureIndex, err := strconv.Atoi(queryDataSlice[3])
// 	if err != nil {
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "解析回调数据错误: " + err.Error(),
// 			ShowAlert:       true,
// 			CacheTime:       60,
// 		})
// 		return nil
// 	}

// 	inputFile, err := utils.GetPicturePhotoInputFile(ctx, serv, cachedArtwork.Pictures[pictureIndex])
// 	if err != nil {
// 		log.Errorf("获取预览图片失败: %s", err)
// 		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
// 			CallbackQueryID: query.ID,
// 			Text:            "获取预览图片失败: " + err.Error(),
// 			ShowAlert:       true,
// 			CacheTime:       3,
// 		})
// 		return nil
// 	}
// 	previewKeyboard := []telego.InlineKeyboardButton{}
// 	if len(cachedArtwork.Pictures) > 1 {
// 		deleteButton := telegoutil.InlineKeyboardButton(fmt.Sprintf("删除这张(%d)", pictureIndex+1)).WithCallbackData("awpv " + dataID + " delete " + strconv.Itoa(pictureIndex) + " " + strconv.Itoa(pictureIndex))
// 		if pictureIndex == 0 {
// 			previewKeyboard = append(previewKeyboard,
// 				deleteButton,
// 				telegoutil.InlineKeyboardButton("下一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", pictureIndex+1, pictureIndex)),
// 			)
// 		} else if pictureIndex == len(cachedArtwork.Pictures)-1 {
// 			previewKeyboard = append(previewKeyboard,
// 				telegoutil.InlineKeyboardButton("上一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", pictureIndex-1, pictureIndex)),
// 				deleteButton,
// 			)
// 		} else {
// 			previewKeyboard = append(previewKeyboard,
// 				telegoutil.InlineKeyboardButton("上一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", pictureIndex-1, pictureIndex)),
// 				deleteButton,
// 				telegoutil.InlineKeyboardButton("下一张").WithCallbackData("awpv "+dataID+fmt.Sprintf(" preview %d %d", pictureIndex+1, pictureIndex)),
// 			)
// 		}
// 	}
// 	postArtworkKeyboard = append(postArtworkKeyboard, previewKeyboard)
// 	msg, err := ctx.Bot().EditMessageMedia(ctx, &telego.EditMessageMediaParams{
// 		ChatID:    callbackMessage.Chat.ChatID(),
// 		MessageID: callbackMessage.MessageID,
// 		Media: telegoutil.MediaPhoto(inputFile).
// 			WithCaption(utils.ArtworkHTMLCaption(meta, cachedArtwork) + fmt.Sprintf("\n<i>当前作品有 %d 张图片</i>", len(cachedArtwork.Pictures))).
// 			WithParseMode(telego.ModeHTML),
// 		ReplyMarkup: telegoutil.InlineKeyboard(
// 			postArtworkKeyboard...,
// 		),
// 	})
// 	if err != nil {
// 		log.Errorf("编辑预览消息失败: %s", err)
// 		return nil
// 	}
// 	cachedArtwork.Pictures[pictureIndex].TelegramInfo.PhotoFileID = msg.Photo[len(msg.Photo)-1].FileID
// 	if err := serv.UpdateCachedArtwork(ctx, cachedArtwork); err != nil {
// 		log.Errorf("更新缓存作品失败: %s", err)
// 	}
// 	return nil
// }
