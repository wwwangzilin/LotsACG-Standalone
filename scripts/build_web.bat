@echo off
REM ============================================================
REM  Build the ManyACG/web frontend into web-dist/ (static SPA)
REM  Usage: scripts\build_web.bat [API_BASE]
REM    API_BASE defaults to /api/v1 (same-origin with the bot REST server)
REM  Requires: git, node (v20+), npm
REM  Output:   web-dist/  (served by the REST server at "/")
REM ============================================================
setlocal
cd /d "%~dp0\.."

set API_BASE=%~1
if "%API_BASE%"=="" set API_BASE=/api/v1

if not exist "web\package.json" (
  echo [web] cloning ManyACG/web ...
  git clone --depth 1 https://github.com/ManyACG/web.git web
  if errorlevel 1 exit /b 1
)

cd web

REM ensure pnpm
where pnpm >nul 2>nul
if errorlevel 1 (
  echo [web] installing pnpm ...
  call npm install -g pnpm@10.24.0
  if errorlevel 1 exit /b 1
)

echo [web] writing .env (API_BASE=%API_BASE%)
(
  echo API_BASE = "%API_BASE%"
  echo NUXT_PUBLIC_BOT_USERNAME =
  echo UMAMI_ID = ""
  echo UMAMI_HOST = ""
  REM CLIENT_MODE=always: 强制浏览器直接请求 API (静态托管, 无 nitro server)
  echo CLIENT_MODE = "always"
) > .env

echo [web] installing dependencies ...
call pnpm install
if errorlevel 1 exit /b 1

echo [web] building (SPA generate) ...
call pnpm exec nuxt generate --config-file nuxt.spa.config.ts
if errorlevel 1 exit /b 1

echo [web] copying output to web-dist ...
if exist "..\web-dist" rmdir /s /q "..\web-dist"
xcopy /e /i /q ".output\public" "..\web-dist" >nul
if errorlevel 1 exit /b 1

echo.
echo [web] Done. Served from web-dist at the REST server root.
echo [web] Remember to set [rest] web_dir = "web-dist" in config.toml
endlocal
