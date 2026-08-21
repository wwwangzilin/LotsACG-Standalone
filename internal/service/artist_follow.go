package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
)

// ArtistFollowData 是某个画师的全局关注数据, 用于增量检测新作品并推送。
type ArtistFollowData struct {
	URL         string   `json:"url"`         // 规范化画师主页链接
	Users       []int64  `json:"users"`       // 关注的用户 ID
	SeenURLs    []string `json:"seen_urls"`   // 已推送过的作品链接 (增量检测基线)
	FirstFollow int64    `json:"first_follow"` // 首次被关注时间 (Unix, 0=未知)
	LastActive  int64    `json:"last_active"`  // 最近一次检测到新作品的时间 (Unix, 0=未知)
	LastAlert   int64    `json:"last_alert"`   // 上次发送「长期无更新」提醒的时间 (Unix, 0=未提醒过)
}

// FollowArtistEntry 是用户维度的关注画师详情 (列表展示用)。
type FollowArtistEntry struct {
	URL      string   `json:"url"`       // 规范化画师主页链接
	Platform string   `json:"platform"`  // 平台 (pixiv/twitter/... 按域名推断)
	Groups   []string `json:"groups"`    // 所属关注分组 (可能为空)
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

// artistFollowGroupsKey 用户关注分组映射: 组名 → 画师 URL 列表。
func artistFollowGroupsKey(userID int64) string {
	return fmt.Sprintf("artistfollow:groups:%d", userID)
}

// ArtistPlatform 按主页链接域名推断画师平台。
func ArtistPlatform(artistURL string) string {
	u := strings.ToLower(strings.TrimSpace(artistURL))
	switch {
	case strings.Contains(u, "pixiv.net"):
		return "pixiv"
	case strings.Contains(u, "twitter.com"), strings.Contains(u, "x.com"):
		return "twitter"
	case strings.Contains(u, "fanbox.cc"), strings.Contains(u, "pixiv.net/fanbox"):
		return "fanbox"
	}
	return "other"
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
		if data.FirstFollow == 0 {
			data.FirstFollow = time.Now().Unix()
		}
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

// FollowArtistWithGroup 关注画师并归入分组 (组名为空等价于 FollowArtist)。
// 返回 added=true 表示本次是新关注。
func (s *Service) FollowArtistWithGroup(ctx context.Context, userID int64, artistURL, group string) (bool, error) {
	added, err := s.FollowArtist(ctx, userID, artistURL)
	if err != nil {
		return false, err
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return added, nil
	}
	groups, _ := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	if groups == nil {
		groups = make(map[string][]string)
	}
	urls := groups[group]
	if !slices.Contains(urls, artistURL) {
		urls = append(urls, artistURL)
		groups[group] = urls
	}
	return added, kvstor.Set(ctx, artistFollowGroupsKey(userID), groups)
}

// ListFollowArtistsDetail 返回用户关注的画师详情 (含平台与所属分组)。
func (s *Service) ListFollowArtistsDetail(ctx context.Context, userID int64) ([]FollowArtistEntry, error) {
	urls, err := s.ListFollowArtists(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, _ := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	entries := make([]FollowArtistEntry, 0, len(urls))
	for _, u := range urls {
		entry := FollowArtistEntry{URL: u, Platform: ArtistPlatform(u)}
		if groups != nil {
			for g, gurls := range groups {
				if slices.Contains(gurls, u) {
					entry.Groups = append(entry.Groups, g)
				}
			}
			slices.Sort(entry.Groups)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// UnfollowArtists 批量取消关注多个画师。返回实际取消的数量。
func (s *Service) UnfollowArtists(ctx context.Context, userID int64, artistURLs []string) (int, error) {
	removed := 0
	for _, u := range artistURLs {
		ok, err := s.UnfollowArtist(ctx, userID, u)
		if err != nil {
			return removed, err
		}
		if ok {
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}
	// 同步清理分组映射中的画师
	groups, _ := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	if groups == nil {
		return removed, nil
	}
	changed := false
	for g, gurls := range groups {
		before := len(gurls)
		gurls = slices.DeleteFunc(gurls, func(u string) bool {
			for _, want := range artistURLs {
				if u == want {
					return true
				}
			}
			return false
		})
		if len(gurls) != before {
			changed = true
			if len(gurls) == 0 {
				delete(groups, g)
			} else {
				groups[g] = gurls
			}
		}
	}
	if changed {
		if err := kvstor.Set(ctx, artistFollowGroupsKey(userID), groups); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// UnfollowArtistGroup 批量取消关注指定分组下的全部画师。返回实际取消的数量。
func (s *Service) UnfollowArtistGroup(ctx context.Context, userID int64, group string) (int, error) {
	group = strings.TrimSpace(group)
	groups, _ := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	if groups == nil {
		return 0, nil
	}
	urls := groups[group]
	if len(urls) == 0 {
		return 0, nil
	}
	removed, err := s.UnfollowArtists(ctx, userID, urls)
	if err != nil {
		return removed, err
	}
	// 组内画师已全部取关 (或本来就没关注), 删除该组
	delete(groups, group)
	_ = kvstor.Set(ctx, artistFollowGroupsKey(userID), groups)
	return removed, nil
}

// ListFollowGroups 返回用户的分组 → 画师 URL 列表。
func (s *Service) ListFollowGroups(ctx context.Context, userID int64) (map[string][]string, error) {
	groups, err := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	if groups == nil {
		return map[string][]string{}, nil
	}
	return groups, nil
}

// SaveFollowGroups 覆盖保存用户的分组映射 (组名 → 画师 URL 列表)。
func (s *Service) SaveFollowGroups(ctx context.Context, userID int64, groups map[string][]string) error {
	if groups == nil {
		groups = make(map[string][]string)
	}
	return kvstor.Set(ctx, artistFollowGroupsKey(userID), groups)
}

// RemoveFollowGroup 删除一个分组 (组内画师仍保持关注, 只移除分组标记)。
func (s *Service) RemoveFollowGroup(ctx context.Context, userID int64, group string) error {
	group = strings.TrimSpace(group)
	groups, err := kvstor.Get[map[string][]string](ctx, artistFollowGroupsKey(userID))
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if _, ok := groups[group]; !ok {
		return oops.Errorf("分组 %q 不存在", group)
	}
	delete(groups, group)
	return kvstor.Set(ctx, artistFollowGroupsKey(userID), groups)
}
