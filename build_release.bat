@echo off
chcp 65001 >nul
setlocal

REM ============================================================
REM  LotsACG Release 鎵撳寘鑴氭湰 (Windows)
REM
REM  鐢ㄦ硶:
REM    build_release.bat              -> 鏋勫缓 + 鎵撳寘 dist (榛樿鐗堟湰 v26.0.0.4)
REM    build_release.bat v26.0.0.4    -> 鎸囧畾鐗堟湰鍙?
REM    build_release.bat v26.0.0.4 publish  -> 鏋勫缓 + 鎵撳寘 + 鐢?gh CLI 鍙戝竷鍒?GitHub
REM
REM  浜х墿:
REM    dist\LotsACG.exe   (鑷洿鏂?/update 涓嬭浇鐨?Windows 鍙墽琛屾枃浠?
REM    dist\LotsACG.zip   (瀹屾暣鍘嬬缉鍖?
REM ============================================================

REM ---- Version: arg %1, default v26.0.0.4 ----
if "%~1"=="" (
    set "VERSION=v26.2.3.0"
) else (
    set "VERSION=%~1"
)
echo [release] version: %VERSION%

REM ---- Git commit ----
git fetch origin >nul 2>&1
set "COMMIT="
for /f "tokens=*" %%i in ('git rev-parse --short origin/HEAD') do set "COMMIT=%%i"
if "%COMMIT%"=="" (
    for /f "tokens=*" %%i in ('git rev-parse --short HEAD') do set "COMMIT=%%i"
)
if "%COMMIT%"=="" set "COMMIT=unknown"
echo [release] commit: %COMMIT%

REM ---- Build time (PowerShell 5.1 compatible) ----
for /f "usebackq tokens=*" %%i in (`powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')"`) do set "BUILD_TIME=%%i"
echo [release] build_time: %BUILD_TIME%

REM ---- Build ----
echo [release] building...
go build -ldflags "-X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Version=%VERSION%' -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Commit=%COMMIT%' -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.BuildTime=%BUILD_TIME%'" -o LotsACG.exe
if errorlevel 1 goto :fail

REM ---- Package ----
echo [release] packaging...
if not exist dist mkdir dist
copy /Y LotsACG.exe dist\LotsACG.exe >nul
powershell -NoProfile -Command "Compress-Archive -Force -Path 'dist\LotsACG.exe' -DestinationPath 'dist\LotsACG.zip'"
if errorlevel 1 goto :fail

echo [release] package done:
echo   dist\LotsACG.exe  (self-update asset)
echo   dist\LotsACG.zip

REM ---- Optional publish ----
if /I "%~2"=="publish" goto :publish
goto :done

:publish
where gh >nul 2>&1
if errorlevel 1 goto :nogh
echo [release] pushing tag %VERSION% ...
git tag %VERSION% origin/HEAD 2>nul
if errorlevel 1 echo [release] tag %VERSION% already exists, reusing
git push origin %VERSION%
echo [release] creating GitHub release %VERSION% ...
REM 蹇呴』鏄惧紡 --repo (鏈湴鏈?upstream 杩滅▼鏃?gh 浼氳鍒ょ洰鏍囦粨搴?
gh release create %VERSION% "dist\LotsACG.exe" "dist\LotsACG.zip" --repo wwwangzilin/LotsACG --title "%VERSION%" --notes "Release %VERSION%"
if errorlevel 1 goto :upload
echo [release] published %VERSION%
goto :done

:upload
echo [release] release may already exist, uploading assets...
gh release upload %VERSION% "dist\LotsACG.exe" "dist\LotsACG.zip" --repo wwwangzilin/LotsACG --clobber
goto :done

:nogh
echo.
echo [release] GitHub CLI (gh) not found, publish manually:
echo   git tag %VERSION% origin/HEAD ^&^& git push origin %VERSION%
echo   gh release create %VERSION% "dist\LotsACG.exe" "dist\LotsACG.zip" --repo wwwangzilin/LotsACG --title "%VERSION%" --notes "Release %VERSION%"
goto :done

:fail
echo [release] BUILD OR PACKAGE FAILED
exit /b 1

:done
echo.
echo [release] done! After publishing, run /update in Telegram.
endlocal
