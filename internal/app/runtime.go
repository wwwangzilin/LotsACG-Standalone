package app

import (
	"context"
	"net/http"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/aiapi"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/database"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/eventbus"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/imseek"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/search"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/tagging"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest"
	restcommon "github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/scheduler"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/kmua"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/repo"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/xppusher"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
)

type Runtime struct {
	cfg     runtimecfg.Config
	closer  func() error
	repos   repo.Repositories
	search  search.Searcher
	service *service.Service
	poster  scheduler.ArtworkPoster
	tgbot   restcommon.TelegramBot
}

func NewRuntime(ctx context.Context, cfg runtimecfg.Config) (*Runtime, error) {
	closer, err := infra.Init(ctx, cfg)
	if err != nil {
		return nil, err
	}

	dbRepo := database.Default()
	searcher := search.Default(ctx)

	// local feature-point search(Optional)
	imCfg := cfg.Imseek
	eng, err := imseek.Init(ctx, imseek.Config{
		Enable:           imCfg.Enable,
		DataDir:          imCfg.DataDir,
		Distance:         imCfg.Distance,
		Count:            imCfg.Count,
		K:                imCfg.K,
		NProbe:           imCfg.NProbe,
		NFeatures:        imCfg.NFeatures,
		MaxHeight:        imCfg.MaxHeight,
		MaxWidth:         imCfg.MaxWidth,
		AutoBuild:        imCfg.AutoBuild,
		BuildDebounceSec: imCfg.BuildDebounceSec,
		MinMatches:       imCfg.MinMatches,
		MinScore:         imCfg.MinScore,
	})
	if err != nil {
		log.Error("imseek init failed, continuing without feature search", "err", err)
		eng, _ = imseek.Init(ctx, imseek.DefaultConfig())
	}
	if eng != nil && eng.Enabled() {
		log.Info("imseek feature search enabled", "data_dir", imCfg.DataDir)
		oldCloser := closer
		closer = func() error {
			_ = eng.Close()
			if oldCloser != nil {
				return oldCloser()
			}
			return nil
		}
	}

	var artworkBus *eventbus.EventBus[*dtoArtworkEventItem]
	needBus := search.Enabled() || (eng != nil && eng.Enabled())
	repos := repo.Repositories(dbRepo)
	if needBus {
		artworkBus = eventbus.New[*dtoArtworkEventItem]()
		if search.Enabled() {
			registerArtworkEventSearcherHandlers(ctx, artworkBus, searcher)
		}
		repos = repo.NewWithArtworkEventImpl(dbRepo, artworkBus)
	}

	serv := service.NewService(
		repos,
		searcher,
		tagging.Default(),
		storage.Storages(),
		source.Sources(),
		cfg.Storage,
		service.WithImseek(eng),
		service.WithAIAPI(aiapi.New(cfg.AIAPI, cfg.XPAIAPI)),
	)
	if artworkBus != nil {
		registerArtworkEventImseekHandlers(ctx, artworkBus, serv)
	}

	return &Runtime{
		cfg:     cfg,
		closer:  closer,
		repos:   repos,
		search:  searcher,
		service: serv,
	}, nil
}

func (r *Runtime) Start(ctx context.Context, stop func()) error {
	if !r.cfg.Telegram.Disable {
		botapp, err := telegram.Init(ctx, r.service, r.cfg.Telegram, r.cfg.App.Debug)
		if err != nil {
			return err
		}
		go botapp.Run(ctx, r.service)
		r.poster = botapp
		r.tgbot = botapp
		// 画师关注 & 标签订阅监控 (watch_interval>0 时启用)
		if r.cfg.Scheduler.WatchInterval > 0 {
			go scheduler.StartFollowWatcher(ctx, r.service, botapp, time.Duration(r.cfg.Scheduler.WatchInterval)*time.Second)
		}
	}

	if r.cfg.Scheduler.Enable && r.poster != nil {
		go scheduler.StartPosterWithConfig(ctx, r.cfg.Scheduler, r.poster, r.service)
	}

	if r.cfg.Rest.Enable {
		opts := []rest.RestAppOption{}
		if r.tgbot != nil {
			opts = append(opts, rest.WithTelegramBot(r.tgbot))
		}
		restApp, err := rest.New(ctx, r.service, r.cfg.Rest, opts...)
		if err != nil {
			return err
		}
		go func() {
			log.Info("Starting RESTful API server", "addr", r.cfg.Rest.Addr)
			if r.cfg.Rest.WebDir != "" {
				webURL := r.cfg.Rest.PublicURL
				if webURL == "" {
					webURL = "http://" + r.cfg.Rest.Addr
				}
				log.Info("Web frontend is available at", "url", webURL)
			}
			if err := restApp.Run(ctx); err != nil {
				log.Error(err)
				stop()
			}
		}()
	}

	// XP-Pusher (Python) 进程: 自动启动 (源码内嵌进 exe, 无需单独部署)
	if r.cfg.XPPusher.AutoStart {
		mgr, err := xppusher.NewManager(r.cfg.XPPusher)
		if err != nil {
			log.Error("xppusher manager init failed", "err", err)
		} else {
			log.Info("XP-Pusher auto-start enabled", "dir", mgr.RunDir(), "log", mgr.LogPath())
			go func() {
				pid, err := mgr.Start(ctx, nil)
				if err != nil {
					log.Error("xppusher auto-start failed (use /xppusher start to retry)", "err", err)
					return
				}
				log.Info("XP-Pusher auto-started", "pid", pid, "log", mgr.LogPath())
			}()
		}
	}

	// kmua-bot (Python) 进程: 自动启动 (源码内嵌进 exe, 无需单独部署)
	if r.cfg.KMua.AutoStart {
		mgr, err := kmua.NewManager(r.cfg.KMua)
		if err != nil {
			log.Error("kmua manager init failed", "err", err)
		} else {
			log.Info("kmua-bot auto-start enabled", "dir", mgr.RunDir(), "log", mgr.LogPath())
			go func() {
				pid, err := mgr.Start(ctx, nil)
				if err != nil {
					log.Error("kmua auto-start failed (use /kmua start to retry)", "err", err)
					return
				}
				log.Info("kmua-bot auto-started", "pid", pid, "log", mgr.LogPath())
			}()
		}
	}

	return nil
}

func (r *Runtime) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer()
}

func (r *Runtime) Cleanup(ctx context.Context) error {
	return r.service.Cleanup(ctx)
}

func Run(ctx context.Context, cfg runtimecfg.Config, stop func()) error {
	log.SetDefault(log.New(log.Config{
		LogFile:    cfg.Log.FilePath,
		MaxBackups: int(cfg.Log.BackupNum),
	}))
	if cfg.App.Debug {
		go func() {
			log.Info("Start pprof server")
			if err := http.ListenAndServe("localhost:39060", nil); err != nil {
				log.Fatal(err)
			}
		}()
	}

	runtime, err := NewRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			log.Error(err)
		}
	}()

	osutil.SetCacheTTL(time.Duration(cfg.Storage.CacheTTL) * time.Second)
	osutil.SetOnRemoveError(func(path string, err error) {
		log.Error("remove cache file error", "path", path, "err", err)
	})

	if err := runtime.Start(ctx, stop); err != nil {
		return err
	}

	log.Info("LotsACG is running !")
	defer log.Info("Exited.")

	<-ctx.Done()
	cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runtime.Cleanup(cleanCtx)
}
