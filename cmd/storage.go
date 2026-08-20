package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/database"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/query"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/spf13/cobra"
)

var (
	storageBatch    int
	storageOrphan   bool
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Storage consistency check (missing / orphan files)",
	Long:  "Scan database artwork pictures against local storage, report missing files and orphan files.",
	Run: func(cmd *cobra.Command, args []string) {
		StorageCheck(cmd.Context())
	},
}

var storageCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check storage consistency (missing / orphan files)",
	Run: func(cmd *cobra.Command, args []string) {
		StorageCheck(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(storageCmd)
	storageCmd.AddCommand(storageCheckCmd)
	storageCmd.PersistentFlags().IntVarP(&storageBatch, "batch", "b", 1000, "Batch size for scanning artworks")
	storageCmd.PersistentFlags().BoolVarP(&storageOrphan, "orphan", "o", false, "Also report orphan files (files not referenced by database)")
}

// StorageCheck 扫描数据库 artwork 的图片本地存储条目, 检测缺失文件; --orphan 时额外报告孤儿文件。
func StorageCheck(ctx context.Context) {
	cfg := runtimecfg.Get()
	log.SetDefault(log.New(log.Config{}))

	closer, err := infra.Init(ctx, cfg)
	if err != nil {
		log.Fatal("failed to initialize infrastructure", "err", err)
	}
	defer func() {
		if closer != nil {
			_ = closer()
		}
	}()

	db := database.Default()
	start := time.Now()

	// 1) 扫描数据库图片, 检测缺失
	log.Info("Scanning picture storage entries (missing files)...")
	missingTotal := 0
	checked := 0
	offset := 0
	knownPaths := make(map[string]struct{})

	for {
		que := query.ArtworksDB{
			ArtworksFilter: query.ArtworksFilter{R18: shared.R18TypeAll},
			Paginate:       query.Paginate{Limit: storageBatch, Offset: offset},
		}
		artworks, err := db.Artwork().QueryArtworks(ctx, que)
		if err != nil {
			log.Error("failed to query artworks", "err", err)
			break
		}
		if len(artworks) == 0 {
			break
		}
		for _, artwork := range artworks {
			for _, pic := range artwork.Pictures {
				checked++
				info := pic.GetStorageInfo()
				for _, kind := range []struct {
					name string
					det  *shared.StorageDetail
				}{
					{"original", info.Original},
					{"regular", info.Regular},
					{"thumb", info.Thumb},
				} {
					if kind.det == nil || kind.det.IsZero() {
						continue
					}
					if kind.det.Type != shared.StorageTypeLocal {
						continue // 只检查本地存储
					}
					if kind.det.Path == "" {
						continue
					}
					knownPaths[kind.det.Path] = struct{}{}
					if _, err := os.Stat(kind.det.Path); err != nil {
						log.Warn("missing storage file", "picture", pic.ID, "kind", kind.name, "path", kind.det.Path, "err", err)
						missingTotal++
					}
				}
			}
		}
		if len(artworks) < storageBatch {
			break
		}
		offset += storageBatch
	}
	log.Info("Missing file scan done", "checked", checked, "missing", missingTotal)

	// 2) 孤儿文件扫描 (可选, dry-run)
	if storageOrphan {
		log.Info("Scanning orphan files (dry-run)...")
		base := strings.TrimSuffix(cfg.Storage.Local.Path, "/")
		orphanCount := 0
		if base != "" && knownPathsCount(knownPaths) > 0 {
			_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				if _, ok := knownPaths[path]; !ok {
					log.Warn("orphan file (dry-run)", "path", path)
					orphanCount++
				}
				return nil
			})
		} else if base == "" {
			log.Warn("local storage path not configured, skip orphan scan")
		}
		log.Info("Orphan scan done (dry-run)", "orphan", orphanCount)
	}

	log.Info("Storage check completed", "duration", time.Since(start).String())
}

// knownPathsCount 返回已收集的本地存储路径数 (避免误判空 map 导致跳过扫描)。
func knownPathsCount(m map[string]struct{}) int {
	return len(m)
}
