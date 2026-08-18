package service

import (
	"context"
	"fmt"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/aiapi"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/imseek"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/search"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/source"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/tagging"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/repo"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
)

type Service struct {
	repos    repo.Repositories
	searcher search.Searcher
	tagger   tagging.Tagger
	storages map[shared.StorageType]storage.Storage
	sources  map[shared.SourceType]source.ArtworkSource
	storCfg  runtimecfg.StorageConfig
	imseek   imseek.Engine
	aiapi    *aiapi.Client
}

type Option func(*Service)

func WithImseek(e imseek.Engine) Option {
	return func(s *Service) { s.imseek = e }
}

func WithAIAPI(c *aiapi.Client) Option {
	return func(s *Service) { s.aiapi = c }
}

func NewService(
	repos repo.Repositories,
	searcher search.Searcher,
	tagger tagging.Tagger,
	storageMap map[shared.StorageType]storage.Storage,
	sourceMap map[shared.SourceType]source.ArtworkSource,
	storCfg runtimecfg.StorageConfig,
	opts ...Option,
) *Service {
	s := &Service{
		repos:    repos,
		tagger:   tagger,
		searcher: searcher,
		storages: storageMap,
		sources:  sourceMap,
		storCfg:  storCfg,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Imseek returns the feature search engine (may be nil or disabled).
func (s *Service) Imseek() imseek.Engine { return s.imseek }

type serviceCtxKey struct{}

var contextKey = serviceCtxKey{}

func WithContext(ctx context.Context, serv *Service) context.Context {
	return context.WithValue(ctx, contextKey, serv)
}

func FromContext(ctx context.Context) *Service {
	if serv, ok := ctx.Value(contextKey).(*Service); ok {
		return serv
	}
	return nil
}

func MustFromContext(ctx context.Context) *Service {
	serv := FromContext(ctx)
	if serv == nil {
		panic(fmt.Sprintf("service: missing service in context (%T)", ctx))
	}
	return serv
}
