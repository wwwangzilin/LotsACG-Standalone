package handlers

import (
	"context"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
)

// 发布队列状态。
const (
	postQueueStatusPosting   = "posting"
	postQueueStatusCancelled = "cancelled"
	postQueueStatusDone      = "done"
)

// postQueueKey 存储最近一次 /post 发布队列。
const postQueueKey = "postqueue:active"

// PostQueue 是一次 /post 命令产生的发布队列。
type PostQueue struct {
	SourceURLs []string  `json:"source_urls"`
	Published  []string  `json:"published"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func loadPostQueue(ctx context.Context) (*PostQueue, error) {
	q, err := kvstor.Get[PostQueue](ctx, postQueueKey)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func savePostQueue(ctx context.Context, q *PostQueue) error {
	return kvstor.Set(ctx, postQueueKey, q)
}

func clearPostQueue(ctx context.Context) error {
	return kvstor.Delete(ctx, postQueueKey)
}
