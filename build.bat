@echo off
setlocal

REM 从 GitHub 拉取最新提交
git fetch origin >nul 2>&1
for /f "tokens=*" %%i in ('git rev-parse --short origin/HEAD') do set COMMIT=%%i
if "%COMMIT%"=="" (
    for /f "tokens=*" %%i in ('git rev-parse --short HEAD') do set COMMIT=%%i
)
for /f "usebackq tokens=*" %%i in (`powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')"`) do set BUILD_TIME=%%i
set VERSION=v26.0.0.3

go build -ldflags "-X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Version=%VERSION%' -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Commit=%COMMIT%' -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.BuildTime=%BUILD_TIME%'" -o LotsACG.exe
