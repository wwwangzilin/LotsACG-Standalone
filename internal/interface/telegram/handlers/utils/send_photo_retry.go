package utils

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/pkg/mediatool"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/ioutil"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// PhotoFileBuilder 按指定压缩力度构建图片文件与发送参数。
type PhotoFileBuilder func(compressLevel int) (*ioutil.Closer[telego.InputFile], *telego.SendPhotoParams, error)

// SendPhotoWithCompressRetry 发送单张图片; 遇到 Telegram "file is too big" 错误时,
// 逐级加强压缩 (缩小边长 + 降低画质) 后重试上传。
func SendPhotoWithCompressRetry(ctx context.Context, bot *telego.Bot, build PhotoFileBuilder) (*telego.Message, error) {
	levels := len(mediatool.TelegramCompressLevels)
	for level := 0; level < levels; level++ {
		file, params, err := build(level)
		if err != nil {
			return nil, err
		}
		msg, err := bot.SendPhoto(ctx, params)
		_ = file.Close()
		if err != nil {
			if IsFileTooBigError(err) && level < levels-1 {
				log.Warn("file too big, recompressing and retrying", "level", level+1)
				continue
			}
			return nil, err
		}
		return msg, nil
	}
	return nil, oops.New("failed to send photo after recompression")
}
