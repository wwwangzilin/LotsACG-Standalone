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

// ArtistFollowData 是某个画师的全局关注数据, 用于增量检测新作品并推送。
type ArtistFollowData struct {
	URL      string   `json:"url"`       // 规范化画师主页链接
	Users    []int64  `json:"users"`     // 关注的用户 ID
	SeenURLs []string `json:"seen_urls"` // 已推送过的作品链接 (增量检测基线)
}

func artistFollowUserKey(userID int64) string {
	return fmt.Sprintf("artistfollow:user:%d", userID)
}

func artistFollowAllKey() string {
	return "artistfollow:all"
}

func artistFollowDataKey(url string) string {
	return fmt.Sprintf("artistfollow:data:%s", url)
}

// FollowArtist 关注一个画师。返回 added=true 表示本次是新关注。
// 首次关注时会预置已见作品基线, 避免把历史作品全部当新作品推送。
func (s *Service) FollowArtist(ctx context.Context, userID int64, artistURL string) (bool, error) {
	normalized := s.FindArtistPageURL(strings.TrimSpace(artistURL))
	if normalized == "" {
		return false, oops.New("不是有效的画师主页链接")
	}
	artistURL = normalized

	data, err := kvstor.Get[ArtistFollowData](ctx, artistFollowDataKey(artistURL))
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return false, err
	}
	added := false
	if !slices.Contains(data.Users, userID) {
		data.Users = append(data.Users, userID)
		added = true
	}
	// 首次关注: 预置已见列表, 只推送之后的新作
	if len(data.SeenURLs) == 0 {
		if urls, err := s.FetchArtistArtworks(ctx, artistURL, 0); err == nil {
			data.SeenURLs = append(data.SeenURLs, urls...)
		}
	}
	if err := kvstor.Set(ctx, artistFollowDataKey(artistURL), data); err != nil {
		return false, err
	}
	// 全局注册表 & 用户关注列表
	all, _ := kvstor.Get[[]string](ctx, artistFollowAllKey())
	if !slices.Contains(all, artistURL) {
		all = append(all, artistURL)
		_ = kvstor.Set(ctx, artistFollowAllKey(), all)
	}
	userURLs, _ := kvstor.Get[[]string](ctx, artistFollowUserKey(userID))
	if !slices.Contains(userURLs, artistURL) {
		userURLs = append(userURLs, artistURL)
		_ = kvstor.Set(ctx, artistFollowUserKey(userID), userURLs)
	}
	return added, nil
}

// UnfollowArtist 取消关注一个画师。返回 removed=true 表示确实取消了关注。
func (s *Service) UnfollowArtist(ctx context.Context, userID int64, artistURL string) (bool, error) {
	normalized := s.FindArtistPageURL(strings.TrimSpace(artistURL))
	if normalized == "" {
		return false, oops.New("不是有效的画师主页链接")
	}
	artistURL = normalized

	data, err := kvstor.Get[ArtistFollowData](ctx, artistFollowDataKey(artistURL))
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
	// 没有关注者时清理该画师的全局数据
	if len(data.Users) == 0 {
		_ = kvstor.Delete(ctx, artistFollowDataKey(artistURL))
		all, _ := kvstor.Get[[]string](ctx, artistFollowAllKey())
		all = slices.DeleteFunc(all, func(u string) bool { return u == artistURL })
		_ = kvstor.Set(ctx, artistFollowAllKey(), all)
	} else {
		_ = kvstor.Set(ctx, artistFollowDataKey(artistURL), data)
	}
	userURLs, _ := kvstor.Get[[]string](ctx, artistFollowUserKey(userID))
	userURLs = slices.DeleteFunc(userURLs, func(u string) bool { return u == artistURL })
	_ = kvstor.Set(ctx, artistFollowUserKey(userID), userURLs)
	return removed, nil
}

// ListFollowArtists 返回用户关注的画师主页链接列表。
func (s *Service) ListFollowArtists(ctx context.Context, userID int64) ([]string, error) {
	urls, err := kvstor.Get[[]string](ctx, artistFollowUserKey(userID))
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return urls, nil
}

// AllFollowedArtists 返回所有被至少一人关注的画师 (供监控任务使用)。
func (s *Service) AllFollowedArtists(ctx context.Context) ([]ArtistFollowData, error) {
	all, err := kvstor.Get[[]string](ctx, artistFollowAllKey())
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]ArtistFollowData, 0, len(all))
	for _, url := range all {
		if data, err := kvstor.Get[ArtistFollowData](ctx, artistFollowDataKey(url)); err == nil && len(data.Users) > 0 {
			result = append(result, data)
		}
	}
	return result, nil
}

// SaveArtistFollowData 保存画师关注数据 (更新已见作品基线)。
func (s *Service) SaveArtistFollowData(ctx context.Context, data *ArtistFollowData) error {
	if data == nil || data.URL == "" {
		return nil
	}
	return kvstor.Set(ctx, artistFollowDataKey(data.URL), *data)
}
