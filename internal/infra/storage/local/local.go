package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	config "github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/storage"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
)

type Local struct {
	basePath string
}

func Init() {
	if !config.Get().Storage.Local.Enable {
		return
	}
	storage.Register(shared.StorageTypeLocal, func() storage.Storage {
		return &Local{
			basePath: strings.TrimSuffix(config.Get().Storage.Local.Path, "/"),
		}
	})
}

func (l *Local) Init(ctx context.Context) error {
	if err := os.MkdirAll(l.basePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (l *Local) Save(ctx context.Context, r io.Reader, storagePath string) (*shared.StorageDetail, error) {
	storagePath = filepath.Join(l.basePath, storagePath)
	fileBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if err := osutil.MkFile(storagePath, fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	return &shared.StorageDetail{
		Type: shared.StorageTypeLocal,
		Path: storagePath,
	}, nil
}

func (l *Local) GetFile(ctx context.Context, detail shared.StorageDetail) (io.ReadCloser, error) {
	return os.Open(detail.Path)
}

func (l *Local) Delete(ctx context.Context, detail shared.StorageDetail) error {
	return osutil.PurgeFile(detail.Path)
}
