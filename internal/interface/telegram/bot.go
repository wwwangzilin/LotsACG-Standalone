package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/metautil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	telegoapiwrapper "github.com/wwwangzilin/LotsACG-Standalone/pkg/telegoapi"
)

type BotApp struct {
	bot              *telego.Bot
	serv             *service.Service
	meta             *metautil.MetaData
	cfg              runtimecfg.TelegramConfig
	debug            bool
	artworkInfoQueue chan artworkInfoTask
	queueMu          sync.Mutex // 保护持久化队列 (KV) 的读写
}

func (app *BotApp) Bot() *telego.Bot {
	return app.bot
}

// QueueDepth 返回当前发布队列深度（用于可观测性指标）。
func (app *BotApp) QueueDepth() float64 {
	if app.artworkInfoQueue == nil {
		return 0
	}
	return float64(len(app.artworkInfoQueue))
}

// NotifyAdmins 向所有配置的管理员发送文本告警（用于抓取失败等异常通知）。
func (app *BotApp) NotifyAdmins(message string) {
	if app.bot == nil {
		return
	}
	for _, adminID := range app.cfg.Admins {
		msg := telegoutil.Message(
			telegoutil.ID(adminID),
			"⚠️ "+message,
		)
		if _, err := app.bot.SendMessage(context.Background(), msg); err != nil {
			log.Warnf("failed to send admin alert: %v", err)
		}
	}
}

func Init(ctx context.Context, serv *service.Service, cfg runtimecfg.TelegramConfig, debug bool) (*BotApp, error) {
	log.Info("Initing telegram client")
	var err error
	apiUrl := cfg.APIURL
	bot, err := telego.NewBot(
		cfg.BotToken,
		telego.WithLogger(log.New(log.Config{
			Level:     log.LevelError,
			FileLevel: log.LevelError,
			LogFile:   "logs/telegram.log",
		})),
		telego.WithAPIServer(apiUrl),
		telego.WithRequestConstructor(telegoapiwrapper.MultipartRequestConstructor{}),
		telego.WithAPICaller(&telegoapiwrapper.RetryRateLimitCaller{
			Caller:       telegoapi.DefaultFastHTTPCaller,
			MaxAttempts:  cfg.Retry.MaxAttempts,
			ExponentBase: cfg.Retry.ExponentBase,
			StartDelay:   time.Duration(cfg.Retry.StartDelay) * time.Second,
			MaxDelay:     time.Duration(cfg.Retry.MaxDelay) * time.Second,
			RateLimit:    telegoapi.RetryRateLimitWait,
		}),
	)
	if err != nil {
		return nil, oops.Errorf("Error when creating bot: %s", err)
	}
	var channelChatID telego.ChatID
	if cfg.ChatID != 0 && cfg.Username != "" {
		channelChatID = telegoutil.ID(cfg.ChatID)
		channelChatID.Username = cfg.Username
	} else if cfg.ChatID != 0 {
		channelChatID = telegoutil.ID(cfg.ChatID)
	} else if cfg.Username != "" {
		channelChatID = telegoutil.Username(cfg.Username)
	} else {
		return nil, oops.New("Either ChatID or Username must be set in config")
	}
	if channelChatID.ID == 0 || channelChatID.Username == "" {
		chatFull, err := bot.GetChat(ctx, &telego.GetChatParams{ChatID: channelChatID})
		if err != nil {
			return nil, oops.Errorf("Error when getting chat info: %s", err)
		}
		channelChatID.ID = chatFull.ID
		channelChatID.Username = chatFull.Username
	}

	var groupChatID telego.ChatID
	if cfg.GroupID != 0 {
		groupChatID = telegoutil.ID(cfg.GroupID)
	}

	// key: telegram:bot:username:<bot_id>
	// value: bot username without @
	botIdStr := strings.Split(cfg.BotToken, ":")[0]
	botId, err := strconv.Atoi(botIdStr)
	if err != nil {
		return nil, oops.Errorf("Invalid bot token: %s", err)
	}
	key := fmt.Sprintf("telegram:bot:username:%d", botId)

	botUsername, err := kvstor.Get[string](ctx, key)
	if err != nil || botUsername == "" {
		me, err := bot.GetMe(ctx)
		if err != nil {
			log.Fatalf("Error when getting bot info: %s", err)
		}
		botUsername = me.Username
	}
	kvstor.Set(ctx, key, botUsername)

	admins := cfg.Admins
	for _, adminID := range admins {
		_, err := serv.GetAdminByTelegramID(ctx, adminID)
		if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
			log.Warnf("Error when getting admin %d: %s", adminID, err)
			continue
		}
		if err == nil {
			continue
		}
		err = serv.CreateAdmin(ctx, adminID, []shared.Permission{shared.PermissionSudo})
		if err != nil {
			log.Warnf("Error when creating admin %d: %s", adminID, err)
			continue
		}
	}

	go func() {
		sig, err := commandsSignature(cfg)
		if err != nil {
			log.Warnf("Error when calculating commands signature: %s", err)
			return
		}
		sigKey := fmt.Sprintf("telegram:bot:commands:%d", botId)
		oldSig, err := kvstor.Get[string](ctx, sigKey)
		if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
			log.Warnf("Error when getting commands signature: %s", err)
			return
		}
		if sig == oldSig {
			// unchanged, skip
			return
		}
		log.Info("Commands signature changed, updating commands...")
		setCommands(ctx, bot, CommonCommands, &telego.BotCommandScopeDefault{Type: telego.ScopeTypeDefault})

		allCommands := append(CommonCommands, AdminCommands...)
		adminUserIDs, err := serv.GetAdminUserIDs(ctx)
		if err != nil {
			log.Warnf("Error when getting admin user IDs: %s", err)
		} else {
			syncAdminUserCommands(ctx, bot, allCommands, adminUserIDs, groupChatID)
		}
		adminGroupIDs, err := serv.GetAdminGroupIDs(ctx)
		if err != nil {
			log.Warnf("Error when getting admin group IDs: %s", err)
		} else {
			for _, adminID := range adminGroupIDs {
				setCommands(ctx, bot, allCommands, &telego.BotCommandScopeChat{
					Type:   telego.ScopeTypeChat,
					ChatID: telegoutil.ID(adminID),
				})
			}
		}

		err = kvstor.Set(ctx, sigKey, sig)
		if err != nil {
			log.Warnf("Error when setting commands signature: %s", err)
			return
		}
	}()

	metaopts := []metautil.Option{}
	if cfg.GroupID != 0 {
		metaopts = append(metaopts, metautil.WithGroupChatID(groupChatID))
	}
	if len(cfg.ExtraTarget) > 0 {
		for _, extraTarget := range cfg.ExtraTarget {
			if extraTarget.ChatID != 0 {
				metaopts = append(metaopts, metautil.WithR18ChannelChatID(telegoutil.ID(extraTarget.ChatID)))
				break
			}
		}
	}
	metaopts = append(metaopts, metautil.WithBotID(int64(botId)))
	// 白名单 = 配置 admins + allowed_users
	allowedUsers := append([]int64{}, cfg.Admins...)
	allowedUsers = append(allowedUsers, cfg.AllowedUsers...)
	metaopts = append(metaopts, metautil.WithAllowedUsers(allowedUsers))
	meta := metautil.NewMetaData(channelChatID, botUsername, metaopts...)

	artworkInfoQueue := make(chan artworkInfoTask, 100)
	app := &BotApp{
		bot:              bot,
		serv:             serv,
		meta:             meta,
		cfg:              cfg,
		debug:            debug,
		artworkInfoQueue: artworkInfoQueue,
	}

	go app.processArtworkInfoTasks(ctx)

	// 恢复持久化队列中未完成的任务 (意外退出后重启继续)
	go app.restoreArtworkInfoQueue(ctx)

	// 导入配置文件中的发送频道 (仅新增, 不覆盖已通过 /channel 命令管理的频道)
	app.importSendChannelsFromConfig(ctx)

	return app, nil
}

// importSendChannelsFromConfig 把 config.toml [telegram.send_channels] 中定义的
// 发送频道导入 KV。已存在 (chat_id 相同) 的频道会被跳过, 保留运行时修改。
func (app *BotApp) importSendChannelsFromConfig(ctx context.Context) {
	for _, cc := range app.cfg.SendChannels {
		if cc.ChatID == 0 {
			continue
		}
		enabled := true
		if cc.Enabled != nil {
			enabled = *cc.Enabled
		}
		ch := &service.SendChannel{
			ChatID:         cc.ChatID,
			Title:          cc.Title,
			Enabled:        enabled,
			R18Mode:        cc.R18Mode,
			LinkOnly:       cc.LinkOnly,
			IncludeTags:    cc.IncludeTags,
			ExcludeTags:    cc.ExcludeTags,
			IncludeArtists: cc.IncludeArtists,
			ExcludeArtists: cc.ExcludeArtists,
		}
		if _, err := app.serv.GetSendChannel(ctx, cc.ChatID); err == nil {
			continue // 已存在 (含运行时修改), 跳过
		} else if !errors.Is(err, errs.ErrRecordNotFound) {
			log.Warnf("failed to check send channel %d: %s", cc.ChatID, err)
			continue
		}
		if err := app.serv.AddSendChannel(ctx, ch); err != nil {
			log.Warnf("failed to import send channel %d: %s", cc.ChatID, err)
			continue
		}
		log.Infof("imported send channel %d (%s) from config", cc.ChatID, ch.Title)
	}
}

func setCommands(ctx context.Context, bot *telego.Bot, commands []telego.BotCommand, scope telego.BotCommandScope) {
	if err := bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{Commands: commands, Scope: scope}); err != nil {
		log.Warnf("Error when setting commands for %T: %s", scope, err)
	}
}

func syncAdminUserCommands(ctx context.Context, bot *telego.Bot, commands []telego.BotCommand, adminUserIDs []int64, groupChatID telego.ChatID) {
	for _, adminID := range adminUserIDs {
		setCommands(ctx, bot, commands, &telego.BotCommandScopeChat{
			Type:   telego.ScopeTypeChat,
			ChatID: telegoutil.ID(adminID),
		})
		if groupChatID.ID == 0 {
			continue
		}
		setCommands(ctx, bot, commands, &telego.BotCommandScopeChatMember{
			Type:   telego.ScopeTypeChat,
			ChatID: groupChatID,
			UserID: adminID,
		})
	}
}

func (app *BotApp) processArtworkInfoTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping artwork info task processor")
			return
		case task := <-app.artworkInfoQueue:
			// 从持久化队列移除 (视为开始处理)
			app.removeArtworkInfoTask(ctx, task)
			err := utils.SendArtworkInfo(task.ctx, app.bot, app.meta, app.serv, task.SourceURL, telegoutil.ID(task.ChatID), utils.SendArtworkInfoOptions{AppendCaption: task.AppendCaption, HasPermission: true})
			if err != nil {
				log.Errorf("Error when sending artwork info: %s", err)
				// 失败则写回持久化队列, 意外退出后重启可重试
				app.persistArtworkInfoTask(context.Background(), task)
			}
		}
	}
}

// artworkInfoQueueKey 是持久化队列在 KV 中的 key。
const artworkInfoQueueKey = "telegram:artworkinfo:queue"

// persistArtworkInfoTask 把任务追加到持久化队列 (KV)。
// 每次入队都会先写入 KV, 因此即使进程在发送前意外退出, 重启后仍可恢复。
func (app *BotApp) persistArtworkInfoTask(ctx context.Context, task artworkInfoTask) {
	app.queueMu.Lock()
	defer app.queueMu.Unlock()
	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		log.Warnf("failed to load artwork info queue: %s", err)
	}
	tasks = append(tasks, task)
	if err := kvstor.Set(ctx, artworkInfoQueueKey, tasks); err != nil {
		log.Warnf("failed to persist artwork info queue: %s", err)
	}
}

// removeArtworkInfoTask 从持久化队列移除指定任务 (按 sourceUrl+chatID+appendCaption 匹配第一个)。
func (app *BotApp) removeArtworkInfoTask(ctx context.Context, task artworkInfoTask) {
	app.queueMu.Lock()
	defer app.queueMu.Unlock()
	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil {
		if !errors.Is(err, errs.ErrRecordNotFound) {
			log.Warnf("failed to load artwork info queue: %s", err)
		}
		return
	}
	for i, t := range tasks {
		if t.matches(task) {
			tasks = append(tasks[:i], tasks[i+1:]...)
			break
		}
	}
	if err := kvstor.Set(ctx, artworkInfoQueueKey, tasks); err != nil {
		log.Warnf("failed to save artwork info queue: %s", err)
	}
}

// restoreArtworkInfoQueue 启动时把持久化队列中未完成的任务重新入队。
// 在独立 goroutine 中执行, 避免阻塞初始化; 入队会因 channel 空间而自然限流。
func (app *BotApp) restoreArtworkInfoQueue(ctx context.Context) {
	app.queueMu.Lock()
	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil {
		app.queueMu.Unlock()
		if errors.Is(err, errs.ErrRecordNotFound) {
			_ = kvstor.Set(ctx, artworkInfoQueueKey, []artworkInfoTask{})
		} else {
			log.Warnf("failed to load artwork info queue: %s", err)
		}
		return
	}
	// 先清空 KV, 恢复的任务后续处理时会重新移除/写回
	if err := kvstor.Set(ctx, artworkInfoQueueKey, []artworkInfoTask{}); err != nil {
		log.Warnf("failed to clear artwork info queue: %s", err)
	}
	app.queueMu.Unlock()

	if len(tasks) == 0 {
		return
	}
	for _, t := range tasks {
		t.ctx = context.Background()
		app.artworkInfoQueue <- t
	}
	log.Infof("restored %d pending artwork info tasks from persistent queue", len(tasks))
}

func (app *BotApp) Run(ctx context.Context, serv *service.Service) {
	log.Info("Start polling")
	updates, err := app.Bot().UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Offset: -1,
		AllowedUpdates: []string{
			telego.MessageUpdates,
			telego.ChannelPostUpdates,
			telego.CallbackQueryUpdates,
			telego.InlineQueryUpdates,
		},
	})
	if err != nil {
		log.Fatalf("Error when getting updates: %s", err)
	}

	botHandler, err := telegohandler.NewBotHandler(app.Bot(), updates,
		telegohandler.WithErrorHandler(func(ctx *telegohandler.Context, update telego.Update, err error) {
			fields := append([]any{"err", err}, updateLogFields(update)...)
			log.Error("telegram handler error", fields...)
		}),
	)
	if err != nil {
		log.Fatalf("Error when creating bot handler: %s", err)
	}
	go func() {
		<-ctx.Done()
		log.Info("Shutting down telegram bot...")
		stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := botHandler.StopWithContext(stopCtx); err != nil {
			log.Warnf("Error when stopping bot handler: %s", err)
		}
		log.Info("Stopped bot handler")
	}()

	if !app.debug {
		botHandler.Use(telegohandler.PanicRecoveryHandler(func(recovered any) error {
			log.Errorf("Panic recovered: %v", recovered)
			return nil
		}))
	}
	botHandler.Use(messageLogger)

	baseGroup := botHandler.BaseGroup()
	handlers.New(app.meta, serv).Register(baseGroup)
	if err := botHandler.Start(); err != nil {
		log.Fatalf("Error when starting bot handler: %s", err)
	}
}
