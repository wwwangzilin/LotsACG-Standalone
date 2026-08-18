package handlers

import (
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// CancelPost 处理 /cancel 指令: 取消当前正在进行的发布队列, 保留已发布的内容。
func CancelPost(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionPostArtwork) {
		utils.ReplyMessage(ctx, message, "你没有发布作品的权限")
		return nil
	}
	q, err := loadPostQueue(ctx)
	if err != nil {
		utils.ReplyMessage(ctx, message, "当前没有进行中的发布队列")
		return nil
	}
	if q.Status != postQueueStatusPosting {
		utils.ReplyMessage(ctx, message, "当前没有进行中的发布队列")
		return nil
	}
	q.Status = postQueueStatusCancelled
	if err := savePostQueue(ctx, q); err != nil {
		utils.ReplyMessage(ctx, message, "取消失败, 请稍后再试")
		return err
	}
	utils.ReplyMessage(ctx, message, fmt.Sprintf("已取消发布队列 (共 %d 条, 已发布 %d 条, 保留已发布内容)", len(q.SourceURLs), len(q.Published)))
	return nil
}

// CancelAndDeletePost 处理 /cd 指令: 取消发布队列, 并删除本次队列中已发布的内容。
func CancelAndDeletePost(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionPostArtwork) {
		utils.ReplyMessage(ctx, message, "你没有发布作品的权限")
		return nil
	}
	q, err := loadPostQueue(ctx)
	if err != nil || len(q.Published) == 0 {
		utils.ReplyMessage(ctx, message, "当前没有可删除的已发布队列")
		return nil
	}
	q.Status = postQueueStatusCancelled
	_ = savePostQueue(ctx, q)

	deleted := 0
	for _, url := range q.Published {
		aw, err := serv.GetArtworkByURL(ctx, url)
		if err != nil {
			continue
		}
		if err := serv.DeleteArtworkByURL(ctx, url); err != nil {
			log.Errorf("cd: failed to delete artwork %s: %s", url, err)
			continue
		}
		if err := serv.StorageDeleteArtworkFiles(ctx, aw); err != nil {
			log.Errorf("cd: failed to delete artwork files %s: %s", url, err)
		}
		deleted++
	}
	_ = clearPostQueue(ctx)
	utils.ReplyMessage(ctx, message, fmt.Sprintf("已取消并删除本次队列中的 %d 条已发布作品", deleted))
	return nil
}
