package telegram

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
)

// initTestKV 用临时 bbolt 文件初始化 KV。
// 注意 kvstor 是包级单例 (sync.Once), 整个测试进程只初始化一次;
// 因此使用固定路径而非 t.TempDir(), 避免 kvstor 仍占用文件时清理失败。
func initTestKV(t *testing.T) {
	t.Helper()
	path := filepath.Join(os.TempDir(), "lotsacg-queue-test-kv.bbolt")
	kvstor.Init(runtimecfg.KVDBConfig{
		Type:           "bbolt",
		Path:           path,
		Bucket:         "test",
		TTLBucket:      "ttl",
		TTLBatchLimit:  100,
		TTLSweepPeriod: 60,
	})
}

func newTestApp() *BotApp {
	return &BotApp{
		artworkInfoQueue: make(chan artworkInfoTask, 100),
	}
}

// clearQueue 清理持久化队列, 避免测试之间相互污染 (KV 是包级单例)。
func clearQueue(t *testing.T) {
	t.Helper()
	err := kvstor.Delete(context.Background(), artworkInfoQueueKey)
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		t.Fatalf("failed to clear queue: %v", err)
	}
}

func TestArtworkInfoQueuePersistRoundtrip(t *testing.T) {
	initTestKV(t)
	clearQueue(t)
	ctx := context.Background()
	app := newTestApp()

	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "https://pixiv.net/artworks/1", ChatID: 111, AppendCaption: "a"})
	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "https://pixiv.net/artworks/2", ChatID: 222, AppendCaption: "b"})

	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil {
		t.Fatalf("failed to load queue: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("want 2 persisted tasks, got %d", len(tasks))
	}
	if tasks[0].SourceURL != "https://pixiv.net/artworks/1" || tasks[0].ChatID != 111 || tasks[0].AppendCaption != "a" {
		t.Errorf("roundtrip mismatch: %+v", tasks[0])
	}
	if tasks[1].SourceURL != "https://pixiv.net/artworks/2" || tasks[1].ChatID != 222 || tasks[1].AppendCaption != "b" {
		t.Errorf("roundtrip mismatch: %+v", tasks[1])
	}
}

func TestArtworkInfoQueueRemove(t *testing.T) {
	initTestKV(t)
	clearQueue(t)
	ctx := context.Background()
	app := newTestApp()

	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u1", ChatID: 1})
	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u2", ChatID: 2})
	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u1", ChatID: 1}) // 重复

	// 移除第一个 u1/1
	app.removeArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u1", ChatID: 1})

	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil {
		t.Fatalf("failed to load queue: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("want 2 tasks after remove, got %d", len(tasks))
	}
	// 剩下的应为 u2/2 和 u1/1 (只移除第一个匹配)
	if tasks[0].SourceURL != "u2" || tasks[1].SourceURL != "u1" {
		t.Errorf("unexpected remaining: %+v", tasks)
	}
}

func TestArtworkInfoQueueRestore(t *testing.T) {
	initTestKV(t)
	clearQueue(t)
	ctx := context.Background()
	app := newTestApp()

	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u1", ChatID: 1})
	app.persistArtworkInfoTask(ctx, artworkInfoTask{SourceURL: "u2", ChatID: 2})

	app.restoreArtworkInfoQueue(ctx)

	// KV 应已清空
	tasks, err := kvstor.Get[[]artworkInfoTask](ctx, artworkInfoQueueKey)
	if err != nil {
		t.Fatalf("failed to load queue: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("want empty queue after restore, got %d", len(tasks))
	}
	// channel 应收到恢复的任务
	got := make([]artworkInfoTask, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case tk := <-app.artworkInfoQueue:
			got = append(got, tk)
		default:
			t.Fatalf("expected %d restored tasks, only got %d", 2, i)
		}
	}
	if got[0].SourceURL != "u1" || got[1].SourceURL != "u2" {
		t.Errorf("restored order mismatch: %+v", got)
	}
}
