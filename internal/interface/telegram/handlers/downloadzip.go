package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/query"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/pkg/mediatool"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/service"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// DownloadZip 处理 /downloadzip x 指令:
// 将回复消息中被选中的图片及其后共 x 张图片打包为 zip 下载。
// 0 < x < 30, 管理员无限制。
func DownloadZip(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	x := 5
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil {
			utils.ReplyMessage(ctx, message, "参数错误, 请指定要打包的图片数量, 如 /downloadzip 5")
			return nil
		}
		x = v
	}
	isAdmin, _ := serv.IsAdminByTgID(ctx, message.From.ID)
	if !isAdmin && (x <= 0 || x >= 30) {
		utils.ReplyMessage(ctx, message, "x 需在 0~30 之间 (0<x<30), 管理员无限制")
		return nil
	}
	if x <= 0 {
		x = 1
	}

	if message.ReplyToMessage == nil {
		helpText := `
<b>使用 /downloadzip 命令回复一张图片, 将把该图片及其后的 x 张图片打包为 zip 下载</b>

命令语法: /downloadzip x
x: 从被选中的图片开始向后打包的图片数量 (0<x<30, 管理员无限制)
`
		utils.ReplyMessageWithHTML(ctx, message, helpText)
		return nil
	}

	// 识别作品与被选中的图片
	artwork, startIndex, err := identifyArtworkAndPicture(ctx, serv, message.ReplyToMessage)
	if err != nil {
		utils.ReplyMessage(ctx, message, err.Error())
		return nil
	}
	if len(artwork.Pictures) == 0 {
		utils.ReplyMessage(ctx, message, "该作品没有图片可打包")
		return nil
	}
	if startIndex >= len(artwork.Pictures) {
		utils.ReplyMessage(ctx, message, "被选中的图片之后没有更多图片")
		return nil
	}
	end := startIndex + x
	if end > len(artwork.Pictures) {
		end = len(artwork.Pictures)
	}

	msg, err := utils.ReplyMessage(ctx, message, fmt.Sprintf("正在打包第 %d~%d 张图片 (%d 张)...", startIndex+1, end, end-startIndex))
	if err != nil {
		return err
	}

	// 打包 zip
	zipBytes, zipName, count, err := buildZipFromPictures(ctx, serv, artwork, startIndex, end)
	if err != nil {
		ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    msg.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "打包失败: " + err.Error(),
		})
		return nil
	}
	if count == 0 {
		ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    msg.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "没有可打包的图片 (下载失败)",
		})
		return nil
	}

	_, err = ctx.Bot().SendDocument(ctx, telegoutil.Document(message.Chat.ChatID(), telegoutil.FileFromBytes(zipBytes, zipName)))
	if err != nil {
		ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    msg.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "发送 zip 失败: " + err.Error(),
		})
		return nil
	}
	ctx.Bot().DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    msg.Chat.ChatID(),
		MessageID: msg.MessageID,
	})
	return nil
}

// identifyArtworkAndPicture 从回复消息识别作品和被选中的图片索引。
// 优先使用链接; 否则通过图片 phash 匹配数据库中的作品。
func identifyArtworkAndPicture(ctx *telegohandler.Context, serv *service.Service, reply *telego.Message) (*entity.Artwork, int, error) {
	if reply == nil {
		return nil, 0, fmt.Errorf("请回复一条包含图片或作品链接的消息")
	}

	// 1. 优先用回复消息中的作品链接
	if url := utils.FindSourceURLInMessage(serv, reply); url != "" {
		aw, err := serv.GetArtworkByURL(ctx, url)
		if err != nil {
			return nil, 0, fmt.Errorf("获取作品信息失败: %v", err)
		}
		idx := matchPictureIndexByPhash(ctx, serv, reply, aw)
		return aw, idx, nil
	}

	// 2. 通过图片 phash 匹配
	file, err := utils.GetMessagePhotoFile(ctx, reply)
	if err != nil {
		return nil, 0, fmt.Errorf("获取图片文件失败: %v", err)
	}
	hash, err := mediatool.GetImagePhashFromReader(bytes.NewReader(file))
	if err != nil {
		return nil, 0, fmt.Errorf("计算图片指纹失败: %v", err)
	}
	pictures, err := serv.QueryPicturesByPhash(ctx, query.PicturesPhash{
		Input:    hash,
		Limit:    1,
		Distance: 10,
	})
	if err != nil || len(pictures) == 0 {
		return nil, 0, fmt.Errorf("无法在数据库中找到该图片所属的作品")
	}
	pic := pictures[0]
	if pic.Artwork == nil {
		return nil, 0, fmt.Errorf("图片未关联作品")
	}
	return pic.Artwork, int(pic.OrderIndex), nil
}

// matchPictureIndexByPhash 通过 phash 在作品图片中匹配被选中的图片索引; 匹配失败返回 0。
func matchPictureIndexByPhash(ctx *telegohandler.Context, serv *service.Service, reply *telego.Message, aw *entity.Artwork) int {
	file, err := utils.GetMessagePhotoFile(ctx, reply)
	if err != nil {
		return 0
	}
	hash, err := mediatool.GetImagePhashFromReader(bytes.NewReader(file))
	if err != nil {
		return 0
	}
	pictures, err := serv.QueryPicturesByPhash(ctx, query.PicturesPhash{
		Input:    hash,
		Limit:    1,
		Distance: 10,
	})
	if err != nil || len(pictures) == 0 {
		return 0
	}
	pic := pictures[0]
	if pic.ArtworkID == aw.ID {
		return int(pic.OrderIndex)
	}
	return 0
}

// buildZipFromPictures 下载并打包 [start, end) 范围内的图片为 zip。
// zip 文件名包含作品链接和其哈希值。
func buildZipFromPictures(ctx context.Context, serv *service.Service, artwork *entity.Artwork, start, end int) ([]byte, string, int, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pid := artworkIDPart(artwork.SourceURL)
	count := 0
	for i := start; i < end; i++ {
		pic := artwork.Pictures[i]
		if pic == nil || pic.Original == "" {
			continue
		}
		f, _, err := utils.DownloadPixivImageWithFallback(ctx, pic.Original, 2)
		if err != nil {
			log.Warn("downloadzip: failed to download picture", "url", pic.Original, "err", err)
			continue
		}
		data, rerr := os.ReadFile(f.Name())
		f.Close()
		if rerr != nil {
			log.Warn("downloadzip: failed to read downloaded file", "err", rerr)
			continue
		}
		ext := filepath.Ext(pic.Original)
		name := fmt.Sprintf("%s_%d%s", pid, i+1, ext)
		w, err := zw.Create(name)
		if err != nil {
			continue
		}
		if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
			continue
		}
		count++
	}
	if err := zw.Close(); err != nil {
		return nil, "", count, err
	}
	if count == 0 {
		return nil, "", 0, nil
	}

	sum := md5.Sum([]byte(artwork.SourceURL))
	zipName := fmt.Sprintf("%s_%s.zip", pid, hex.EncodeToString(sum[:4]))
	return buf.Bytes(), zipName, count, nil
}

// artworkIDPart 从作品链接提取简短标识用于文件名。
func artworkIDPart(sourceURL string) string {
	trimmed := strings.TrimRight(sourceURL, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx >= 0 && idx < len(trimmed)-1 {
		return trimmed[idx+1:]
	}
	return "artwork"
}
