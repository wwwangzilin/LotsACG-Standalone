package httpclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/imroc/req/v3"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
	"golang.org/x/sync/singleflight"
)

type bytesCounterKey struct{}

// WithBytesCounter 在 context 中携带一个下载字节计数器, 用于统计某次流程(如批量发布)下载的总字节数。
// 未携带计数器的 context 调用下载不受影响。
func WithBytesCounter(ctx context.Context, counter *int64) context.Context {
	return context.WithValue(ctx, bytesCounterKey{}, counter)
}

// addBytes 将 n 字节累加到 context 携带的计数器中 (无计数器时为空操作)。
func addBytes(ctx context.Context, n int64) {
	if counter, ok := ctx.Value(bytesCounterKey{}).(*int64); ok && counter != nil {
		atomic.AddInt64(counter, n)
	}
}

var (
	defaultClient *req.Client
	once          sync.Once
	dlGroup       singleflight.Group
)

func initDefaultClient() {
	c := req.C().
		ImpersonateChrome().
		SetCommonRetryCount(2).
		SetLogger(log.Default()).
		EnableDebugLog()
	defaultClient = c
	if proxyUrl := runtimecfg.Get().HttpClient.Proxy; proxyUrl != "" {
		defaultClient.SetProxyURL(proxyUrl)
	}
}

func getCachePath(url string) string {
	ext, _ := strutil.GetFileExtFromURL(url)
	return filepath.Join(runtimecfg.Get().Storage.CacheDir, "req", strutil.MD5Hash(url)+ext)
}

// DownloadWithCache downloads a file with caching. If the file is already cached, it returns the cached file.
func DownloadWithCache(ctx context.Context, url string, client *req.Client) (
	*osutil.File,
	error,
) {
	once.Do(initDefaultClient)
	if client == nil {
		client = defaultClient
	}
	cachePath := getCachePath(url)
	if fi, err := os.Stat(cachePath); err == nil && !fi.IsDir() {
		addBytes(ctx, fi.Size())
		return osutil.OpenCache(cachePath)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ch := dlGroup.DoChan(cachePath, func() (any, error) {
		if fi, err := os.Stat(cachePath); err == nil && !fi.IsDir() {
			return nil, nil
		}
		resp, err := client.R().
			SetContext(ctx).
			SetOutputFile(cachePath).
			Get(url)
		if err != nil {
			os.Remove(cachePath)
			return nil, err
		}
		if resp.IsErrorState() {
			os.Remove(cachePath)
			return nil, fmt.Errorf("http error: %d", resp.GetStatusCode())
		}
		return nil, nil
	})
	select {
	case r := <-ch:
		if r.Err != nil {
			return nil, r.Err
		}
		if fi, err := os.Stat(cachePath); err == nil {
			addBytes(ctx, fi.Size())
		}
		return osutil.OpenCache(cachePath)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
