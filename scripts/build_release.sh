#!/usr/bin/env bash
# ============================================================
#  LotsACG Release 打包脚本 (Linux / macOS / CI)
#
#  用法:
#    ./scripts/build_release.sh                  -> 构建 + 打包 (默认 v26.0.0.4)
#    ./scripts/build_release.sh v26.0.0.4        -> 指定版本
#    ./scripts/build_release.sh v26.0.0.4 publish -> 构建 + 打包 + gh release
#
#  产物:
#    dist/lotsacg_linux_amd64        (Linux 二进制)
#    dist/lotsacg_linux_amd64.tar.gz
# ============================================================
set -euo pipefail

VERSION="${1:-v26.0.0.4}"
PUBLISH="${2:-}"

echo "[release] version: $VERSION"

# Git commit
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo "[release] commit: $COMMIT"

# Build time (UTC)
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "[release] build_time: $BUILD_TIME"

# Build
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
OUT="lotsacg_${OS}_${ARCH}"
echo "[release] building $OS/$ARCH ..."
CGO_ENABLED=0 go build \
  -ldflags "-X 'github.com/wwwangzilin/LotsACG/internal/common/version.Version=${VERSION}' -X 'github.com/wwwangzilin/LotsACG/internal/common/version.Commit=${COMMIT}' -X 'github.com/wwwangzilin/LotsACG/internal/common/version.BuildTime=${BUILD_TIME}'" \
  -o "dist/${OUT}" .

echo "[release] packaging ..."
mkdir -p dist
tar -czf "dist/${OUT}.tar.gz" -C dist "${OUT}"

echo "[release] done:"
echo "  dist/${OUT}"
echo "  dist/${OUT}.tar.gz"

if [ "${PUBLISH}" = "publish" ]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "[release] gh CLI not found, skip publishing."
    echo "  gh release create ${VERSION} \"dist/${OUT}\" \"dist/${OUT}.tar.gz\" --title \"${VERSION}\" --notes \"Release ${VERSION}\""
  else
    echo "[release] creating GitHub release ${VERSION} ..."
    gh release create "${VERSION}" "dist/${OUT}" "dist/${OUT}.tar.gz" --title "${VERSION}" --notes "Release ${VERSION}" \
      || gh release upload "${VERSION}" "dist/${OUT}" "dist/${OUT}.tar.gz" --clobber
  fi
fi

echo "[release] done!"
