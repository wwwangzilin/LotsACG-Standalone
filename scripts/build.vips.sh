#!/bin/bash
set -e

builtAt="$(date +'%F %T %z')"
# 优先使用 GitHub 最新提交, 失败则回退到本地 HEAD
git fetch origin >/dev/null 2>&1
gitCommit=$(git rev-parse --short origin/HEAD 2>/dev/null || git log --pretty=format:"%h" -1)
version=$(git describe --abbrev=0 --tags 2>/dev/null || echo "v26.0.0.1")

versionFlags="-w -s \
-X 'github.com/wwwangzilin/LotsACG/internal/common/version.BuildTime=$builtAt' \
-X 'github.com/wwwangzilin/LotsACG/internal/common/version.Commit=$gitCommit' \
-X 'github.com/wwwangzilin/LotsACG/internal/common/version.Version=$version'"

vipsFlags=$(pkg-config --static --libs vips)

# nodynamic tag is for https://github.com/gen2brain/avif
CGO_ENABLED=1 go build \
    -tags vips,nodynamic,netgo \
    -ldflags "$versionFlags -linkmode external -extldflags \"-static $vipsFlags\"" \
    -o lotsacg
