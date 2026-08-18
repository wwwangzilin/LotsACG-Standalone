package pixiv

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/imroc/req/v3"
	"github.com/samber/oops"
)

// respBodyBytes 返回响应体原始字节, 并手动处理 gzip/brotli 解压。
//
// 背景: Go 的 http.Transport 只有在"自己"添加 Accept-Encoding 头时才会自动解压。
// 由于我们手动设置了 Accept-Encoding: gzip (且 ImpersonateChrome 会设置 br),
// Transport 不会自动解压, 需要在此手动处理。
func respBodyBytes(resp *req.Response) ([]byte, error) {
	body := resp.Bytes()
	if len(body) < 2 {
		return body, nil
	}
	// gzip 魔数: 0x1f 0x8b
	if body[0] == 0x1f && body[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, oops.Wrapf(err, "gzip init failed")
		}
		defer zr.Close()
		out, err := io.ReadAll(zr)
		if err != nil {
			return nil, oops.Wrapf(err, "gzip read failed")
		}
		return out, nil
	}
	// brotli 魔数: 0xce 0xb2 0xcf 0x81
	if len(body) >= 4 && body[0] == 0xce && body[1] == 0xb2 && body[2] == 0xcf && body[3] == 0x81 {
		br := brotli.NewReader(bytes.NewReader(body))
		out, err := io.ReadAll(br)
		if err != nil {
			return nil, oops.Wrapf(err, "brotli read failed")
		}
		return out, nil
	}
	return body, nil
}
