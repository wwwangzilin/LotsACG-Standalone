package telegoapi

import (
	"time"

	"github.com/mymmrac/telego/telegoapi"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/http/httpproxy"
)

// NewFastHTTPCaller 创建带可选代理的 fasthttp caller。
// proxyAddr 为空时直连；支持 http/https/socks5 代理（如 http://127.0.0.1:7890、socks5://127.0.0.1:1080）。
func NewFastHTTPCaller(proxyAddr string) *telegoapi.FastHTTPCaller {
	client := &fasthttp.Client{
		MaxConnsPerHost:               20,
		MaxIdleConnDuration:           90 * time.Second,
		ReadTimeout:                   60 * time.Second,
		WriteTimeout:                  60 * time.Second,
		MaxResponseBodySize:           100 * 1024 * 1024, // 100MB，Telegram 大文件
		DisableHeaderNamesNormalizing: true,
	}
	if proxyAddr != "" {
		dialer := &fasthttpproxy.Dialer{
			Timeout:       10 * time.Second,
			ConnectTimeout: 10 * time.Second,
			Config: httpproxy.Config{
				HTTPProxy:  proxyAddr,
				HTTPSProxy: proxyAddr,
			},
		}
		if dialFunc, err := dialer.GetDialFunc(false); err == nil {
			client.Dial = dialFunc
		}
	}
	return &telegoapi.FastHTTPCaller{Client: client}
}
