package service

import (
	"context"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/httpclient"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// ShouldAIAutoTag 是否启用了 AI 自动打标 (新作品入库时)。
func (s *Service) ShouldAIAutoTag() bool {
	if s.aiapi == nil || !s.aiapi.Enabled() {
		return false
	}
	cfg := runtimecfg.Get()
	return cfg.AIAPI.AutoTag || cfg.XPAIAPI.AutoTag
}

// AutoTagCachedArtwork 为新入库作品自动打标:
//  1. 若图片 tagger 已启用 (tagging.tagnew), 逐图推理标签;
//  2. 若 AI 自动打标已启用 (aiapi.auto_tag), 根据标题与已有标签补充 AI 标签。
//
// 结果合并进 artwork.Tags 并持久化到 cached artwork。
func (s *Service) AutoTagCachedArtwork(ctx context.Context, artwork *entity.CachedArtworkData) error {
	if artwork == nil {
		return nil
	}
	merged := make(map[string]struct{}, len(artwork.Tags))
	for _, t := range artwork.Tags {
		if t != "" {
			merged[t] = struct{}{}
		}
	}

	// 1. 图片 tagger
	if s.ShouldTagNewArtwork() {
		log.Info("auto tagging: predicting tags from pictures", "url", artwork.SourceURL)
		for i, pic := range artwork.Pictures {
			err := func() error {
				if detail := pic.StorageInfo.Original; detail != nil {
					file, err := s.StorageGetFile(ctx, *detail)
					if err != nil {
						return err
					}
					defer file.Close()
					result, err := s.tagger.Predict(ctx, file)
					if err != nil {
						return err
					}
					for tag := range result {
						if tag != "" {
							merged[tag] = struct{}{}
						}
					}
					return nil
				}
				file, err := httpclient.DownloadWithCache(ctx, pic.Original, nil)
				if err != nil {
					return err
				}
				defer file.Close()
				result, err := s.tagger.Predict(ctx, file)
				if err != nil {
					return err
				}
				for tag := range result {
					if tag != "" {
						merged[tag] = struct{}{}
					}
				}
				return nil
			}()
			if err != nil {
				log.Error("auto tagging: failed to predict tags for picture", "err", err, "index", i, "url", pic.Original)
			}
		}
	}

	// 2. AI 标签
	if s.ShouldAIAutoTag() {
		log.Info("auto tagging: generating AI tags", "url", artwork.SourceURL)
		baseTags := make([]string, 0, len(merged))
		for t := range merged {
			baseTags = append(baseTags, t)
		}
		aiTags, err := s.aiapi.GenerateTags(ctx, artwork.Title, baseTags)
		if err != nil {
			log.Error("auto tagging: AI tag generation failed", "err", err, "url", artwork.SourceURL)
		} else {
			for _, t := range aiTags {
				if t != "" {
					merged[t] = struct{}{}
				}
			}
		}
	}

	if len(merged) == len(artwork.Tags) {
		return nil // 没有新增标签
	}
	newTags := make([]string, 0, len(merged))
	for t := range merged {
		newTags = append(newTags, t)
	}
	artwork.Tags = newTags
	if err := s.UpdateCachedArtwork(ctx, artwork); err != nil {
		log.Warn("auto tagging: failed to update cached artwork", "err", err)
		return err
	}
	log.Info("auto tagging: merged tags", "url", artwork.SourceURL, "total", len(newTags))
	return nil
}
