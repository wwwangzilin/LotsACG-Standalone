#!/usr/bin/env bash
# ============================================================
#  Build the ManyACG/web frontend into web-dist/ (static SPA)
#  Usage: scripts/build_web.sh [API_BASE]
#    API_BASE defaults to /api/v1 (same-origin with the bot REST server)
#  Requires: git, node (v20+), npm
#  Output:   web-dist/  (served by the REST server at "/")
# ============================================================
set -euo pipefail
cd "$(dirname "$0")/.."

API_BASE="${1:-/api/v1}"

if [ ! -f "web/package.json" ]; then
  echo "[web] cloning ManyACG/web ..."
  git clone --depth 1 https://github.com/ManyACG/web.git web
fi

cd web

# ensure pnpm
if ! command -v pnpm >/dev/null 2>&1; then
  echo "[web] installing pnpm ..."
  npm install -g pnpm@10.24.0
fi

echo "[web] writing .env (API_BASE=${API_BASE})"
cat > .env <<EOF
API_BASE = "${API_BASE}"
NUXT_PUBLIC_BOT_USERNAME =
UMAMI_ID = ""
UMAMI_HOST = ""
# CLIENT_MODE=always: 强制浏览器直接请求 API (静态托管, 无 nitro server)
CLIENT_MODE = "always"
EOF

echo "[web] installing dependencies ..."
pnpm install

echo "[web] building (SPA generate) ..."
pnpm exec nuxt generate --config-file nuxt.spa.config.ts

echo "[web] copying output to web-dist ..."
rm -rf ../web-dist
mkdir -p ../web-dist
cp -r .output/public/. ../web-dist/

echo ""
echo "[web] Done. Served from web-dist at the REST server root."
echo "[web] Remember to set [rest] web_dir = \"web-dist\" in config.toml"
