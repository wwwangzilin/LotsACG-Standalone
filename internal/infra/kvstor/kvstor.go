package kvstor

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/redis/rueidis"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"go.etcd.io/bbolt"
)

type KVStore interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// GetRaw 返回 key 对应的原始 msgpack 字节; 不存在或已过期返回 errs.ErrRecordNotFound。
	GetRaw(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

var (
	defaultDb  KVStore
	initOnce   sync.Once
	reaperStop chan struct{}
)

// openBolt 打开 bbolt 数据库，自动创建父目录（解决全新环境下 data 目录不存在导致启动失败）。
func openBolt(dbPath string) (*bbolt.DB, error) {
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return bbolt.Open(dbPath, 0600, nil)
}

func Init(cfg runtimecfg.KVDBConfig) {
	switch cfg.Type {
	case "bbolt":
		dbPath := cfg.Path
		initOnce.Do(func() {
			bdb, err := openBolt(dbPath)
			if err != nil {
				log.Fatal("Failed to initialize kvdb", "err", err)
			}
			reaperStop = make(chan struct{})
			bbdb := &bboltDB{
				db:             bdb,
				bucket:         cfg.Bucket,
				ttlBucket:      cfg.TTLBucket,
				ttlBatchLimit:  cfg.TTLBatchLimit,
				ttlSweepPeriod: time.Duration(cfg.TTLSweepPeriod) * time.Second,
				stop:           reaperStop,
			}
			defaultDb = bbdb
			bbdb.startTTLReaper()
			startKVBoltBackup(bbdb, cfg)
		})
	case "redis":
		initOnce.Do(func() {
			var (
				client rueidis.Client
				err    error
			)
			rc := cfg.Redis
			if rc.URL != "" {
				opt := rueidis.MustParseURL(rc.URL)
				if rc.TLS {
					opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: rc.TLSInsecure}
				}
				client, err = rueidis.NewClient(opt)
			} else {
				opt := rueidis.ClientOption{
					InitAddress: rc.Addrs,
					Username:    rc.Username,
					Password:    rc.Password,
					SelectDB:    rc.DB,
				}
				if rc.TLS {
					opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: rc.TLSInsecure}
				}
				client, err = rueidis.NewClient(opt)
			}
			if err != nil {
				log.Fatal("Failed to initialize redis kvdb", "err", err)
			}
			defaultDb = &redisDB{client: client, prefix: rc.Prefix}
		})
	default:
		dbPath := cfg.Path
		initOnce.Do(func() {
			bdb, err := openBolt(dbPath)
			if err != nil {
				log.Fatal("Failed to initialize kvdb", "err", err)
			}
			reaperStop = make(chan struct{})
			bbdb := &bboltDB{
				db:             bdb,
				bucket:         cfg.Bucket,
				ttlBucket:      cfg.TTLBucket,
				ttlBatchLimit:  cfg.TTLBatchLimit,
				ttlSweepPeriod: time.Duration(cfg.TTLSweepPeriod) * time.Second,
				stop:           reaperStop,
			}
			defaultDb = bbdb
			bbdb.startTTLReaper()
			startKVBoltBackup(bbdb, cfg)
		})
	}

}

// startKVBoltBackup 根据配置启动 bbolt 定期备份。
func startKVBoltBackup(bbdb *bboltDB, cfg runtimecfg.KVDBConfig) {
	if cfg.BackupDir == "" {
		return
	}
	interval := time.Duration(cfg.BackupInterval) * time.Second
	if cfg.BackupInterval == 0 {
		interval = 24 * time.Hour
	}
	keep := cfg.BackupKeep
	if keep == 0 {
		keep = 7
	}
	bbdb.startBackup(cfg.BackupDir, interval, keep)
}

func Close() error {
	if defaultDb != nil {
		if stop := reaperStop; stop != nil {
			close(stop)
		}
		return defaultDb.Close()
	}
	return nil
}

func Set(ctx context.Context, key string, value any) error {
	return defaultDb.Set(ctx, key, value, 0)
}

// SetWithTTL stores the value with a given TTL; a non-positive TTL behaves like Set.
func SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		return Set(ctx, key, value)
	}
	return defaultDb.Set(ctx, key, value, ttl)
}

// Get 读取并 msgpack 解码到具体类型 T。
// 注意: 之前实现在底层把值反序列化成 any 再做类型断言,
// 对 struct/map/指针 会得到 map[string]interface{} 导致断言永远失败,
// 因此改为直接在原始字节上解码到 T。
func Get[T any](ctx context.Context, key string) (T, error) {
	var zero T
	raw, err := defaultDb.GetRaw(ctx, key)
	if err != nil {
		return zero, err
	}
	if err := msgpack.Unmarshal(raw, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func Delete(ctx context.Context, key string) error {
	return defaultDb.Delete(ctx, key)
}
