package handlers

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v3"
	"github.com/imroc/req/v3"
	"github.com/unvgo/ouid"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source/impls/pixiv"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/common"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/rest/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
	"gorm.io/datatypes"
)

func HandleGetPictureFileByID(ctx fiber.Ctx) error {
	requestCtx := ctx.RequestCtx()
	safeCtx := ctx.Context()
	pictureID := ctx.Params("id")
	if pictureID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing picture ID")
	}
	id, err := ouid.FromObjectIDHex(pictureID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid picture ID")
	}
	serv := common.MustGetState[*service.Service](ctx, common.StateKeyService)
	picture, err := serv.GetPictureByID(requestCtx, id)
	if err != nil {
		return err
	}
	var filePath string
	if detail := picture.StorageInfo.Data().Original; detail != nil {
		file, err := serv.StorageGetFile(requestCtx, *detail)
		if err != nil {
			return err
		}
		defer file.Close()
		filePath = file.Name()
	} else {
		file, err := httpclient.DownloadWithCache(safeCtx, picture.Original, nil)
		if err != nil {
			return err
		}
		defer file.Close()
		filePath = file.Name()
	}
	ctx.Set(fiber.HeaderContentDisposition, "inline; filename=\""+serv.PrettyFileName(picture.Artwork, picture)+"\"")
	return ctx.SendFile(filePath, fiber.SendFile{Compress: true})
}

func HandleGetSizedPictureFileByID(ctx fiber.Ctx) error {
	requestCtx := ctx.RequestCtx()
	size := ctx.Params("size")
	if !slices.Contains([]string{"thumb", "regular", "original"}, size) {
		size = "regular"
	}
	pictureID := ctx.Params("id")
	if pictureID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing picture ID")
	}
	id, err := ouid.FromObjectIDHex(pictureID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid picture ID")
	}
	serv := common.MustGetState[*service.Service](ctx, common.StateKeyService)
	picture, err := serv.GetPictureByID(requestCtx, id)
	if errors.Is(err, errs.ErrRecordNotFound) {
		return common.NewError(fiber.StatusNotFound, "picture not found")
	}
	if err != nil {
		return err
	}
	var detail *shared.StorageDetail
	switch size {
	case "thumb":
		detail = picture.StorageInfo.Data().Thumb
	case "regular":
		detail = picture.StorageInfo.Data().Regular
	case "original":
		detail = picture.StorageInfo.Data().Original
	}
	if detail != nil {
		if detail.Mime != "" {
			ctx.Set(fiber.HeaderContentType, detail.Mime)
			var sendErr error
			sendWriter := func(w *bufio.Writer) {
				defer w.Flush()
				sendErr = serv.StorageStreamFile(requestCtx, *detail, w)
			}
			ctx.SendStreamWriter(sendWriter)
			return sendErr
		}

		pr, pw := io.Pipe()
		streamCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := make(chan error, 1)
		go func() {
			defer close(errChan)
			err := serv.StorageStreamFile(streamCtx, *detail, pw)
			if err != nil {
				pw.CloseWithError(err)
				errChan <- err
				return
			}
			pw.Close()
		}()

		buf := make([]byte, 3072)
		n, err := io.ReadAtLeast(pr, buf, 3072)
		if err != nil {
			cancel()
			pr.Close()

			streamErr := <-errChan
			if streamErr != nil {
				return streamErr
			}

			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return err
			}
		}

		buf = buf[:n]
		mtype := mimetype.Detect(buf)
		if mtype != nil {
			ctx.Set(fiber.HeaderContentType, mtype.String())
			go func() {
				// 异步更新 mime 类型
				detail.Mime = mtype.String()
				storData := picture.StorageInfo.Data()
				if size == "thumb" {
					storData.Thumb = detail
				} else if size == "regular" {
					storData.Regular = detail
				} else if size == "original" {
					storData.Original = detail
				}
				picture.StorageInfo = datatypes.NewJSONType(storData)
				if err := serv.SavePicture(context.Background(), picture); err != nil {
					log.Errorf("failed to save picture mime: %v", err)
				}
			}()
		}
		ctx.Set(fiber.HeaderContentDisposition, "inline")

		fullReader := io.MultiReader(bytes.NewReader(buf), pr)
		return ctx.SendStream(fullReader)
	}
	// 无存储信息: 本地下载并返回 (避免依赖外部图床, 修复图片外链不可达时的加载失败)
	// 按优先级尝试多个图源: 配置代理 -> pixiv.cat -> i.muxmus.com -> 官方 i.pximg.net
	safeCtx := ctx.Context()
	proxyHosts := runtimecfg.Get().Source.Pixiv.ImgProxyHosts()
	candidates := pixiv.BuildPixivImageCandidates(picture.Original, proxyHosts)
	client := buildPixivDownloadClient()
	var file *osutil.File
	var dlErr error
	for _, candidate := range candidates {
		file, dlErr = httpclient.DownloadWithCache(safeCtx, candidate, client)
		if dlErr == nil {
			break
		}
		log.Warnf("download picture %s via %s failed: %v", picture.ID, candidate, dlErr)
	}
	if dlErr != nil {
		// 全部失败: 回退到缩略图外链
		return ctx.Redirect().To(picture.Thumbnail)
	}
	defer file.Close()
	ctx.Set(fiber.HeaderContentDisposition, "inline; filename=\""+serv.PrettyFileName(picture.Artwork, picture)+"\"")
	return ctx.SendFile(file.Name(), fiber.SendFile{Compress: true})
}

// buildPixivDownloadClient 构建带 pixiv cookie/Referer 的下载客户端 (直连优先)。
// 代理降级由 httpclient.DownloadWithCache 内部处理 (直连失败自动走 proxyClient)。
func buildPixivDownloadClient() *req.Client {
	cfg := runtimecfg.Get()
	cookies := make([]*http.Cookie, 0, len(cfg.Source.Pixiv.Cookies))
	for _, ck := range cfg.Source.Pixiv.Cookies {
		if ck.Name != "" && ck.Value != "" {
			cookies = append(cookies, &http.Cookie{Name: ck.Name, Value: ck.Value})
		}
	}
	client := req.C().ImpersonateChrome().
		SetCommonCookies(cookies...).
		SetCommonHeaders(map[string]string{"Referer": "https://www.pixiv.net/"}).
		SetCommonRetryCount(1)
	return client
}

func HandleGetRandomPicture(ctx fiber.Ctx) error {
	serv := common.MustGetState[*service.Service](ctx, common.StateKeyService)
	picture, err := serv.RandomPictures(ctx.RequestCtx(), 1)
	if err != nil {
		return err
	}
	if len(picture) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "no picture found")
	}
	pic := picture[0]
	if pic.StorageInfo.Data() == shared.ZeroStorageInfo || pic.StorageInfo.Data().Regular == nil {
		return ctx.Redirect().To(pic.Thumbnail)
	}
	cfg := common.MustGetState[runtimecfg.RestConfig](ctx, common.StateKeyConfig)
	picUrl := utils.ResponseUrlForStoragePath(ctx, *pic.GetStorageInfo().Regular, cfg.StoragePathRules)
	if picUrl == "" {
		return ctx.Redirect().To(pic.Thumbnail)
	}
	return ctx.Redirect().To(picUrl)
}
