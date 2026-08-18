package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/samber/oops"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/common/version"
	config "github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/interface/telegram/handlers/utils"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

const (
	updateRepoSlug  = "wwwangzilin/LotsACG"
	updateApiLatest = "https://api.github.com/repos/" + updateRepoSlug + "/releases/latest"
)

// ghAsset GitHub release 资产。
type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ghRelease GitHub release 信息。
type ghRelease struct {
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Assets  []ghAsset `json:"assets"`
}

// githubHTTPClient 返回带代理的 HTTP 客户端 (用于访问 GitHub API 与下载)。
func githubHTTPClient() *http.Client {
	// 优先使用 [source] proxy, 其次 [telegram] proxy
	proxyURL := config.Get().Source.Proxy
	if proxyURL == "" {
		proxyURL = config.Get().Telegram.Proxy
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Timeout: 5 * time.Minute, Transport: transport}
}

// checkLatestRelease 查询 GitHub 最新 release。
func checkLatestRelease(ctx context.Context) (*ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", updateApiLatest, nil)
	if err != nil {
		return nil, oops.Wrapf(err, "create github request failed")
	}
	req.Header.Set("User-Agent", "LotsACG")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := githubHTTPClient().Do(req)
	if err != nil {
		return nil, oops.Wrapf(err, "github api request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, oops.Errorf("github api http error: %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, oops.Wrapf(err, "github api unmarshal failed")
	}
	return &rel, nil
}

// parseVersion 解析版本号 (兼容 v 前缀与 4 段版本号, 如 26.5.2.5)。
// semver 标准只支持 major.minor.patch, 这里把第 4 段作为 prerelease 数字:
// 26.5.2.5 -> 26.5.2-5 (同 patch 时按数值比较, 26.5.2.5 > 26.5.2.0 成立)。
func parseVersion(s string) (semver.Version, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	if len(parts) == 4 {
		s = strings.Join(parts[:3], ".") + "-" + parts[3]
	}
	return semver.Parse(s)
}

// latestVersionName 返回 release 的版本名 (优先 Name, 回退 TagName)。
func (r *ghRelease) latestVersionName() string {
	if r.Name != "" {
		return r.Name
	}
	return r.TagName
}

// pickAsset 选择当前平台的安装资产。
func (r *ghRelease) pickAsset() *ghAsset {
	if runtime.GOOS == "windows" {
		// 优先 .exe
		for i := range r.Assets {
			if strings.HasSuffix(strings.ToLower(r.Assets[i].Name), ".exe") {
				return &r.Assets[i]
			}
		}
		// 其次含 windows 的压缩包
		for i := range r.Assets {
			if strings.Contains(strings.ToLower(r.Assets[i].Name), "windows") {
				return &r.Assets[i]
			}
		}
	}
	// 其他平台: 匹配 goos 或通用压缩包
	goos := strings.ToLower(runtime.GOOS)
	for i := range r.Assets {
		name := strings.ToLower(r.Assets[i].Name)
		if strings.Contains(name, goos) || strings.Contains(name, runtime.GOARCH) {
			return &r.Assets[i]
		}
	}
	if len(r.Assets) > 0 {
		return &r.Assets[0]
	}
	return nil
}

// UpdateCmd 处理 /update 指令: 检查 GitHub 最新 release 并提供更新确认。
func UpdateCmd(ctx *telegohandler.Context, message telego.Message) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionInGroup(ctx, serv, message, shared.PermissionSudo) {
		utils.ReplyMessage(ctx, message, "你没有执行更新的权限")
		return nil
	}
	rel, err := checkLatestRelease(ctx)
	if err != nil {
		utils.ReplyMessage(ctx, message, "检查更新失败: "+err.Error())
		return nil
	}
	latestName := rel.latestVersionName()
	cur, err := parseVersion(version.Version)
	if err != nil {
		cur = semver.Version{}
	}
	latest, err := parseVersion(latestName)
	if err != nil {
		utils.ReplyMessage(ctx, message, "无法解析最新版本号: "+latestName)
		return nil
	}

	if !latest.GT(cur) {
		utils.ReplyMessage(ctx, message, fmt.Sprintf("当前已是最新版本 <code>%s</code> (最新 release: %s)", version.Version, latestName))
		return nil
	}

	asset := rel.pickAsset()
	if asset == nil {
		utils.ReplyMessage(ctx, message, fmt.Sprintf("发现新版本 <code>%s</code>, 但未找到匹配当前平台的安装包", latestName))
		return nil
	}
	sizeMB := float64(asset.Size) / 1024 / 1024
	oldExeName := strings.TrimSuffix(filepath.Base(mustExecutable()), filepath.Ext(mustExecutable())) + ".old.exe"
	reply, _ := utils.ReplyMessageWithHTML(ctx, message, fmt.Sprintf(
		"发现新版本 <code>%s</code> (当前 <code>%s</code>)\n安装包: <code>%s</code> (%.1f MB)\n\n确认后会自动下载并重启, 旧版本会归档为 <code>%s</code>, 更新失败时自动回退。",
		latestName, version.Version, asset.Name, sizeMB, oldExeName,
	))
	if reply == nil {
		return nil
	}
	_, err = ctx.Bot().EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{
		ChatID:    telegoutil.ID(reply.Chat.ID),
		MessageID: reply.MessageID,
		ReplyMarkup: telegoutil.InlineKeyboard(telegoutil.InlineKeyboardRow(
			telegoutil.InlineKeyboardButton("✅ 确认更新").WithCallbackData("update_confirm"),
			telegoutil.InlineKeyboardButton("取消").WithCallbackData("update_cancel"),
		)),
	})
	if err != nil {
		log.Warn("update: failed to set confirm markup", "err", err)
	}
	return nil
}

// UpdateConfirmCallback 处理更新确认: 下载并执行自更新。
func UpdateConfirmCallback(ctx *telegohandler.Context, query telego.CallbackQuery) error {
	serv, err := requireService(ctx)
	if err != nil {
		return err
	}
	if !utils.CheckPermissionForQuery(ctx, serv, query, shared.PermissionSudo) {
		ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID, Text: "没有权限", ShowAlert: true, CacheTime: 60})
		return nil
	}
	if query.Message == nil {
		return nil
	}
	msg := query.Message
	edit := func(text string) {
		_, _ = ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:      msg.GetChat().ChatID(),
			MessageID:   msg.GetMessageID(),
			Text:        text,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: nil,
		})
	}

	rel, err := checkLatestRelease(ctx)
	if err != nil {
		edit("检查更新失败: " + err.Error())
		return nil
	}
	asset := rel.pickAsset()
	if asset == nil {
		edit("未找到匹配当前平台的安装包")
		return nil
	}
	edit(fmt.Sprintf("正在下载 <code>%s</code>...", asset.Name))

	exePath := mustExecutable()
	exeDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)
	exeBase := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	newPath := filepath.Join(exeDir, exeBase+"_new.exe")
	lastPct := int64(-1)
	if err := downloadToFile(ctx, asset.BrowserDownloadURL, newPath, asset.Size, func(done, total int64) {
		if total <= 0 {
			return
		}
		pct := done * 100 / total
		// 每 5% 或完成时更新进度, 避免频繁 EditMessage 触发 Telegram 限流
		if pct >= lastPct+5 || done >= total {
			lastPct = pct
			edit(fmt.Sprintf("⏳ 正在下载 <code>%s</code>... %d%% (%d/%d MB)", asset.Name, pct, done/1024/1024, total/1024/1024))
		}
	}); err != nil {
		_ = os.Remove(newPath)
		edit("下载更新失败: " + err.Error())
		return nil
	}
	edit("下载完成, 正在准备重启...")

	if runtime.GOOS == "windows" {
		if err := writeUpdateBat(exeDir, exeName); err != nil {
			edit("生成更新脚本失败: " + err.Error())
			return nil
		}
		// 启动更新脚本 (detached), 脚本等待当前进程退出后替换并重启。
		// 注意: 不用 `start /min "" "path"` 嵌套 (Go 拼命令行时的引号转义会让
		// start 解析错乱, 导致 "Windows 找不到文件"), 直接用 cmd /c 执行 bat,
		// DETACHED_PROCESS 分离 + Release 不阻塞当前进程。
		batPath := filepath.Join(exeDir, "update.bat")
		cmd := exec.Command("cmd.exe", "/c", batPath)
		cmd.Dir = exeDir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008} // DETACHED_PROCESS
		if err := cmd.Start(); err != nil {
			edit("启动更新脚本失败: " + err.Error())
			return nil
		}
		_ = cmd.Process.Release()
		edit("✅ 更新已开始, bot 即将自动重启 (几秒后)")
		// 给 Telegram 一点时间发出消息, 然后退出自身让脚本接管
		go func() {
			time.Sleep(3 * time.Second)
			os.Exit(0)
		}()
		return nil
	}
	edit("当前平台暂不支持自动更新, 请手动替换二进制")
	return nil
}

// UpdateCancelCallback 取消更新。
func UpdateCancelCallback(ctx *telegohandler.Context, query telego.CallbackQuery) error {
	if query.Message != nil {
		_, _ = ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:      query.Message.GetChat().ChatID(),
			MessageID:   query.Message.GetMessageID(),
			Text:        "已取消更新",
			ReplyMarkup: nil,
		})
	}
	return nil
}

// mustExecutable 返回当前可执行文件路径, 失败时返回空字符串。
func mustExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		log.Warn("update: cannot get executable path", "err", err)
		return "."
	}
	return exe
}

// downloadToFile 下载文件到指定路径, 并校验大小。progress 可选 (done, total 字节)。
func downloadToFile(ctx context.Context, downloadURL, dest string, expectedSize int64, progress func(done, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return oops.Wrapf(err, "create download request failed")
	}
	req.Header.Set("User-Agent", "LotsACG")
	resp, err := githubHTTPClient().Do(req)
	if err != nil {
		return oops.Wrapf(err, "download request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return oops.Errorf("download http error: %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return oops.Wrapf(err, "create file failed")
	}
	defer out.Close()

	// 循环读写以便报告进度
	buf := make([]byte, 256*1024)
	var total int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return oops.Wrapf(werr, "write file failed")
			}
			total += int64(n)
			if progress != nil {
				progress(total, expectedSize)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return oops.Wrapf(rerr, "read response failed")
		}
	}

	// 校验: 下载内容不能太小 (避免下载到错误页面), 且尽量接近期望大小
	if total < 1024*1024 {
		return oops.Errorf("downloaded file too small (%d bytes)", total)
	}
	if expectedSize > 0 && total < expectedSize/2 {
		return oops.Errorf("downloaded file size mismatch (got %d, want %d)", total, expectedSize)
	}
	return nil
}

// writeUpdateBat 生成 Windows 更新脚本: 等旧进程退出 → 归档旧 exe → 替换 → 启动新 exe。
// exeName 为实际运行的 exe 文件名 (如 LotsACG.exe), 不硬编码, 保证任意改名都能更新。
func writeUpdateBat(exeDir, exeName string) error {
	exeBase := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	oldName := exeBase + ".old.exe"
	newName := exeBase + "_new.exe"
	bat := `@echo off
chcp 65001 >nul
rem cd to exe dir so relative paths work regardless of cwd
cd /d "%~dp0"
timeout /t 3 /nobreak >nul
:kill
taskkill /IM __EXE__ /F >nul 2>&1
timeout /t 1 /nobreak >nul
if exist __OLD__ del /F __OLD__
move /Y __EXE__ __OLD__ >nul 2>&1
if errorlevel 1 goto kill
move /Y __NEW__ __EXE__ >nul 2>&1
if errorlevel 1 goto restore
echo [LotsACG] update success
start "" __EXE__
del "%~f0"
exit /b 0
:restore
move /Y __OLD__ __EXE__ >nul 2>&1
echo [LotsACG] update failed, restored old version (__OLD__)
del "%~f0"
exit /b 1
`
	bat = strings.ReplaceAll(bat, "__EXE__", exeName)
	bat = strings.ReplaceAll(bat, "__NEW__", newName)
	bat = strings.ReplaceAll(bat, "__OLD__", oldName)
	// bat 需要 CRLF 换行
	bat = strings.ReplaceAll(bat, "\n", "\r\n")
	return os.WriteFile(filepath.Join(exeDir, "update.bat"), []byte(bat), 0755)
}
