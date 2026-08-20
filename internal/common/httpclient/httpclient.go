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
	proxyClient   *req.Client
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

	// 代理客户端: 仅用于直连失败时的降级重试, 避免图片代理域名可直连时也消耗代理流量。
	// 代理来源: httpclient.proxy 优先, 其次 source.proxy (主人的 Clash 配置位置)。
	proxyUrl := runtimecfg.Get().HttpClient.Proxy
	if proxyUrl == "" {
		proxyUrl = runtimecfg.Get().Source.Proxy
	}
	if proxyUrl != "" {
		pc := req.C().
			ImpersonateChrome().
			SetCommonRetryCount(2).
			SetLogger(log.Default()).
			EnableDebugLog()
		pc.SetProxyURL(proxyUrl)
		proxyClient = pc
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
		// 直连优先: 图片代理域名 (manyacg/i.pixiv.re 等) 可直连时不走代理, 省流量
		dlClient := client
		if dlClient == nil {
			dlClient = defaultClient
		}
		resp, err := dlClient.R().
			SetContext(ctx).
			SetOutputFile(cachePath).
			Get(url)
		if err == nil && !resp.IsErrorState() {
			return nil, nil
		}
		if err != nil {
			log.Warnf("download direct failed, will retry via proxy: %v", err)
		} else {
			log.Warnf("download direct http error %d, will retry via proxy", resp.GetStatusCode())
		}
		os.Remove(cachePath)

		// 直连失败: 降级到代理重试 (若配置了代理)
		if proxyClient != nil {
			resp2, err2 := proxyClient.R().
				SetContext(ctx).
				SetOutputFile(cachePath).
				Get(url)
			if err2 != nil {
				os.Remove(cachePath)
				return nil, err2
			}
			if resp2.IsErrorState() {
				os.Remove(cachePath)
				return nil, fmt.Errorf("http error via proxy: %d", resp2.GetStatusCode())
			}
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("http error: %d", resp.GetStatusCode())
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
