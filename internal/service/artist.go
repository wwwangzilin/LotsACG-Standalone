package service

import (
	"context"
	"fmt"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/unvgo/ouid"
)

func (s *Service) GetArtistByID(ctx context.Context, id ouid.OUID) (*entity.Artist, error) {
	return s.repos.Artist().GetArtistByID(ctx, id)
}

// ListAllArtists 分页遍历返回全部作者 (供批量订阅等场景)。
func (s *Service) ListAllArtists(ctx context.Context) ([]entity.Artist, error) {
	const pageSize = 200
	all := make([]entity.Artist, 0)
	offset := 0
	for {
		page, err := s.repos.Artist().ListArtists(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		offset += len(page)
	}
	return all, nil
}

// ArtistPageURL 根据作者实体构造规范化主页链接; 不支持的源返回空串。
func ArtistPageURL(a *entity.Artist) string {
	if a == nil {
		return ""
	}
	switch a.Type {
	case shared.SourceTypePixiv:
		if a.UID != "" {
			return fmt.Sprintf("https://www.pixiv.net/users/%s", a.UID)
		}
	}
	return ""
}

// SubscribeAllArtists 一键订阅所有能构造主页链接的作者 (跳过已订阅)。
// 返回本次新增订阅数量。
func (s *Service) SubscribeAllArtists(ctx context.Context, userID int64) (int, error) {
	artists, err := s.ListAllArtists(ctx)
	if err != nil {
		return 0, err
	}
	subscribed := 0
	for i := range artists {
		url := ArtistPageURL(&artists[i])
		if url == "" {
			continue
		}
		added, err := s.FollowArtist(ctx, userID, url)
		if err != nil {
			continue // 单个失败不中断
		}
		if added {
			subscribed++
		}
	}
	return subscribed, nil
}
