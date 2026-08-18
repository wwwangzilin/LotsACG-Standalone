package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
)

// TagSubscribeData 是某个标签的全局订阅数据, 用于增量检测新作品并推送。
type TagSubscribeData struct {
	Tag      string   `json:"tag"`       // 订阅的标签
	Users    []int64  `json:"users"`     // 订阅的用户 ID
	SeenURLs []string `json:"seen_urls"` // 已推送过的作品链接 (增量检测基线)
}

func tagSubUserKey(userID int64) string {
	return fmt.Sprintf("tagsub:user:%d", userID)
}

func tagSubAllKey() string {
	return "tagsub:all"
}

func tagSubDataKey(tag string) string {
	return fmt.Sprintf("tagsub:data:%s", tag)
}

// SubscribeTag 订阅一个标签。返回 added=true 表示本次是新订阅。
// 首次订阅时会预置最新作品作为基线, 只推送之后的新作品。
func (s *Service) SubscribeTag(ctx context.Context, userID int64, tag string) (bool, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false, oops.New("标签不能为空")
	}
	data, err := kvstor.Get[TagSubscribeData](ctx, tagSubDataKey(tag))
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return false, err
	}
	added := false
	if !slices.Contains(data.Users, userID) {
		data.Users = append(data.Users, userID)
		added = true
	}
	// 首次订阅: 搜索当前最新作品作为基线, 避免把历史作品全部当新作品推送
	if len(data.SeenURLs) == 0 {
		if arts, err := s.SearchNewArtworksByTagsOrdered(ctx, []string{tag}, 10, "date_d"); err == nil {
			for _, a := range arts {
				if a != nil && a.SourceURL != "" {
					data.SeenURLs = append(data.SeenURLs, a.SourceURL)
				}
			}
		}
	}
	if err := kvstor.Set(ctx, tagSubDataKey(tag), data); err != nil {
		return false, err
	}
	// 全局注册表 & 用户订阅列表
	all, _ := kvstor.Get[[]string](ctx, tagSubAllKey())
	if !slices.Contains(all, tag) {
		all = append(all, tag)
		_ = kvstor.Set(ctx, tagSubAllKey(), all)
	}
	userTags, _ := kvstor.Get[[]string](ctx, tagSubUserKey(userID))
	if !slices.Contains(userTags, tag) {
		userTags = append(userTags, tag)
		_ = kvstor.Set(ctx, tagSubUserKey(userID), userTags)
	}
	return added, nil
}

// UnsubscribeTag 取消订阅一个标签。返回 removed=true 表示确实取消了订阅。
func (s *Service) UnsubscribeTag(ctx context.Context, userID int64, tag string) (bool, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false, oops.New("标签不能为空")
	}
	data, err := kvstor.Get[TagSubscribeData](ctx, tagSubDataKey(tag))
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	removed := false
	data.Users = slices.DeleteFunc(data.Users, func(id int64) bool {
		if id == userID {
			removed = true
			return true
		}
		return false
	})
	if len(data.Users) == 0 {
		_ = kvstor.Delete(ctx, tagSubDataKey(tag))
		all, _ := kvstor.Get[[]string](ctx, tagSubAllKey())
		all = slices.DeleteFunc(all, func(t string) bool { return t == tag })
		_ = kvstor.Set(ctx, tagSubAllKey(), all)
	} else {
		_ = kvstor.Set(ctx, tagSubDataKey(tag), data)
	}
	userTags, _ := kvstor.Get[[]string](ctx, tagSubUserKey(userID))
	userTags = slices.DeleteFunc(userTags, func(t string) bool { return t == tag })
	_ = kvstor.Set(ctx, tagSubUserKey(userID), userTags)
	return removed, nil
}

// ListSubscribedTags 返回用户订阅的标签列表。
func (s *Service) ListSubscribedTags(ctx context.Context, userID int64) ([]string, error) {
	tags, err := kvstor.Get[[]string](ctx, tagSubUserKey(userID))
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return tags, nil
}

// AllSubscribedTags 返回所有被至少一人订阅的标签 (供监控任务使用)。
func (s *Service) AllSubscribedTags(ctx context.Context) ([]TagSubscribeData, error) {
	all, err := kvstor.Get[[]string](ctx, tagSubAllKey())
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]TagSubscribeData, 0, len(all))
	for _, tag := range all {
		if data, err := kvstor.Get[TagSubscribeData](ctx, tagSubDataKey(tag)); err == nil && len(data.Users) > 0 {
			result = append(result, data)
		}
	}
	return result, nil
}

// SaveTagSubscribeData 保存标签订阅数据 (更新已见作品基线)。
func (s *Service) SaveTagSubscribeData(ctx context.Context, data *TagSubscribeData) error {
	if data == nil || data.Tag == "" {
		return nil
	}
	return kvstor.Set(ctx, tagSubDataKey(data.Tag), *data)
}
