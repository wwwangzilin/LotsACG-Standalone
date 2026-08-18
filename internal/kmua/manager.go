package kmua

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const pidKey = "kmua:pid"

// kmuaCodeVersion 内嵌 kmua-bot 源码版本标记。
// 修改 internal/kmua/project 下的源码时需同步递增, 触发重新提取覆盖。
const kmuaCodeVersion = "20260816.1"

// Manager 管理内嵌的 kmua-bot (Python) 进程。
// 日志统一写入 <exe>/logs/kmua.log, 与 LotsACG 日志同目录。
// kmua-bot 依赖复杂 (Python 3.13 + git 源依赖), 环境准备用 uv (uv sync)。
type Manager struct {
	mu     sync.Mutex
	cfg    runtimecfg.KMuaConfig
	exeDir string
}

func NewManager(cfg runtimecfg.KMuaConfig) (*Manager, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, exeDir: filepath.Dir(exe)}, nil
}

// RunDir 返回 kmua-bot 实际运行目录: 显式配置 dir 优先, 否则为 <exeDir>/kmua (内嵌提取)。
func (m *Manager) RunDir() string {
	if m.cfg.Dir != "" {
		return m.cfg.Dir
	}
	return filepath.Join(m.exeDir, "kmua")
}

// LogPath 返回 kmua-bot 日志文件路径 (与 LotsACG 日志同目录)。
func (m *Manager) LogPath() string {
	return filepath.Join(m.exeDir, "logs", "kmua.log")
}

// EnsureExtracted 确保 kmua-bot 源码就绪:
// 显式 dir -> 校验 pyproject.toml 存在; 否则把内嵌源码提取到 <exeDir>/kmua。
// 版本感知: 内嵌源码版本变化时重新覆盖源码, 但保留 settings.toml / data (数据库)。
func (m *Manager) EnsureExtracted() error {
	if m.cfg.Dir != "" {
		if _, err := os.Stat(filepath.Join(m.cfg.Dir, "pyproject.toml")); err != nil {
			return fmt.Errorf("kmua 目录中没有 pyproject.toml: %s", m.cfg.Dir)
		}
		return nil
	}
	dir := m.RunDir()
	marker := filepath.Join(dir, ".kmua_version")
	needExtract := false
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err != nil {
		needExtract = true
	} else if data, err := os.ReadFile(marker); err != nil || strings.TrimSpace(string(data)) != kmuaCodeVersion {
		needExtract = true
		log.Info("kmua: embedded source updated, re-extracting", "dir", dir)
	}
	if !needExtract {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	err := fs.WalkDir(ProjectFS, "project", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(path, "project"), "/")
		if rel == "" {
			return nil
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		data, err := ProjectFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	})
	if err != nil {
		return err
	}
	_ = os.WriteFile(marker, []byte(kmuaCodeVersion), 0644)
	log.Info("kmua: embedded source extracted", "dir", dir)
	return nil
}

// EnsureSettings 确保 settings.toml 就绪:
// 已存在则不覆盖; 否则用外部 Settings 文件或内嵌模板, 并用 [kmua] 配置段覆盖
// token / owners / webapp 等字段后写出。
func (m *Manager) EnsureSettings() error {
	dir := m.RunDir()
	dst := filepath.Join(dir, "settings.toml")
	if _, err := os.Stat(dst); err == nil {
		return nil // 已存在, 保留用户已有配置
	}
	var template []byte
	var err error
	if m.cfg.Settings != "" {
		template, err = os.ReadFile(m.cfg.Settings)
	} else {
		template, err = ProjectFS.ReadFile("project/settings.toml")
	}
	if err != nil {
		return err
	}
	var s map[string]any
	if err := toml.Unmarshal(template, &s); err != nil {
		// 模板解析失败时直接写模板 (dynaconf 能读)
		return os.WriteFile(dst, template, 0644)
	}
	if m.cfg.Token != "" {
		s["token"] = m.cfg.Token
	}
	if len(m.cfg.Owners) > 0 {
		s["owners"] = m.cfg.Owners
	}
	if m.cfg.Proxy != "" {
		s["proxy"] = m.cfg.Proxy
	}
	if m.cfg.Webapp {
		s["webapp"] = true
	}
	if m.cfg.WebappPort != 0 {
		s["webapp_port"] = m.cfg.WebappPort
	}
	if m.cfg.WebappURL != "" {
		s["webapp_url"] = m.cfg.WebappURL
	}
	if m.cfg.WebappShortName != "" {
		s["webapp_short_name"] = m.cfg.WebappShortName
	}
	out, err := toml.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, out, 0644); err != nil {
		return err
	}
	log.Info("kmua: settings.toml generated", "dir", dir)
	return nil
}

// uvPath 返回可用的 uv 可执行文件 (显式配置优先, 否则 PATH)。
func (m *Manager) uvPath() string {
	if m.cfg.UV != "" {
		return m.cfg.UV
	}
	if commandExists("uv") {
		return "uv"
	}
	return ""
}

// venvPython 返回 kmua-bot venv 中的 Python 解释器路径。
func venvPython(dir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(dir, ".venv", "bin", "python")
}

// PrepareEnv 准备运行环境并返回 Python 解释器路径:
// 显式配置 python -> venv python (已装好) -> uv sync 创建 venv 并安装依赖。
func (m *Manager) PrepareEnv(ctx context.Context, progress func(string)) (string, error) {
	dir := m.RunDir()
	if m.cfg.Python != "" {
		if !pythonCanImport(m.cfg.Python, "kmua") {
			return "", fmt.Errorf("配置的 python 无法导入 kmua: %s", m.cfg.Python)
		}
		return m.cfg.Python, nil
	}
	venvPy := venvPython(dir)
	if pythonCanImport(venvPy, "kmua") {
		return venvPy, nil
	}
	uv := m.uvPath()
	if uv == "" {
		return "", errors.New("未找到 uv (kmua 依赖复杂, 需用 uv 创建环境), 请安装 uv 或在 [kmua] 配置 uv 路径")
	}
	if progress != nil {
		progress("⏳ 正在用 uv 创建 kmua 环境并安装依赖 (首次需要几分钟, 需联网)...")
	}
	// Windows 无 uvloop / tgcrypto / pyturso (都只有 sdist, 编译需 MSVC/Rust):
	// uvloop 已 patch 为可选 (__main__.py try/except); tgcrypto 对 kurigram 是可选加速;
	// pyturso (agentfs-sdk 依赖) 仅在 agent 功能开启时导入 (默认 false), 所以跳过安装。
	// numpy: uv.lock 锁定的 1.26.4 无 py3.13 wheel, pyproject 已强制 >=2.0.0。
	args := []string{"sync", "--no-dev",
		"--no-install-package", "tgcrypto",
		"--no-install-package", "uvloop",
		"--no-install-package", "pyturso"}
	env := append(os.Environ(), "PYTHONUTF8=1")
	// uv 不认 --proxy 命令行参数, 用环境变量设置代理
	if proxy := runtimeProxy(); proxy != "" {
		env = append(env,
			"HTTP_PROXY="+proxy,
			"HTTPS_PROXY="+proxy,
			"ALL_PROXY="+proxy,
		)
	}
	if err := runCommandEnv(uv, dir, env, args...); err != nil {
		// 部分失败 (如个别包无 wheel) 但环境可用时继续
		if pythonCanImport(venvPy, "kmua") {
			log.Warn("kmua: uv sync 有包安装失败, 但环境已可用 (继续启动)")
			return venvPy, nil
		}
		return "", fmt.Errorf("uv sync 安装依赖失败: %w", err)
	}
	if !pythonCanImport(venvPy, "kmua") {
		return "", errors.New("uv sync 完成但 venv 中无法导入 kmua")
	}
	return venvPy, nil
}

// Start 启动 kmua-bot 进程 (日志写入 logs/kmua.log)。
func (m *Manager) Start(ctx context.Context, progress func(string)) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pid := m.RunningPID(ctx); pid != 0 {
		return pid, nil
	}
	if err := m.EnsureExtracted(); err != nil {
		return 0, err
	}
	if err := m.EnsureSettings(); err != nil {
		return 0, err
	}
	dir := m.RunDir()
	python, err := m.PrepareEnv(ctx, progress)
	if err != nil {
		return 0, err
	}
	command := m.cfg.Command
	if command == "" {
		command = "-m kmua"
	}
	args := strings.Fields(command)
	if m.cfg.Args != "" {
		args = append(args, strings.Fields(m.cfg.Args)...)
	}
	if err := os.MkdirAll(filepath.Dir(m.LogPath()), 0755); err != nil {
		return 0, err
	}
	logFile, err := os.OpenFile(m.LogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	execCmd := exec.Command(python, args...)
	execCmd.Dir = dir
	// 日志同时写入文件 (供 /kmua log) 与 LotsACG 控制台 (实时滚动, 带 [kmua] 前缀)
	execCmd.Stdout = newTeeWriter("[kmua] ", os.Stdout, logFile)
	execCmd.Stderr = newTeeWriter("[kmua] ", os.Stderr, logFile)
	execCmd.Env = append(os.Environ(),
		"PYTHONUTF8=1",
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
	)
	if runtime.GOOS == "windows" {
		execCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008} // DETACHED_PROCESS
	}
	if err := execCmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, err
	}
	// 注意: tee 持续引用 logFile, 不能立即关闭。进程退出后再由 goroutine 关闭。
	go func() {
		_ = execCmd.Wait()
		_ = logFile.Close()
	}()
	_ = kvstor.Set(ctx, pidKey, execCmd.Process.Pid)
	log.Info("kmua: started", "pid", execCmd.Process.Pid, "dir", dir, "python", python, "log", m.LogPath())
	return execCmd.Process.Pid, nil
}

// Stop 停止 kmua-bot 进程。
func (m *Manager) Stop(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pid := m.RunningPID(ctx)
	if pid == 0 {
		_ = kvstor.Delete(ctx, pidKey)
		return 0, nil
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run(); err != nil {
			return pid, err
		}
	} else if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
	_ = kvstor.Delete(ctx, pidKey)
	log.Info("kmua: stopped", "pid", pid)
	return pid, nil
}

// RunningPID 返回正在运行的 kmua-bot 进程 PID, 0 表示未运行。
func (m *Manager) RunningPID(ctx context.Context) int {
	pid, err := kvstor.Get[int](ctx, pidKey)
	if err != nil || pid <= 0 {
		return 0
	}
	if !processAlive(pid) {
		_ = kvstor.Delete(ctx, pidKey)
		return 0
	}
	return pid
}

// LogTail 返回 kmua-bot 日志最后 n 行。
func (m *Manager) LogTail(n int) string {
	data, err := os.ReadFile(m.LogPath())
	if err != nil {
		return "(暂无日志: " + err.Error() + ")"
	}
	if n <= 0 {
		n = 30
	}
	rawLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		lines = append(lines, decodeLogLine(line))
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// decodeLogLine 按行解码日志: 合法 UTF-8 直接用, 否则按 GB18030 转码。
func decodeLogLine(line string) string {
	if utf8.ValidString(line) {
		return line
	}
	if s, err := simplifiedchinese.GB18030.NewDecoder().String(line); err == nil {
		return s
	}
	return line
}

func processAlive(pid int) bool {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), strconv.Itoa(pid))
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func pythonCanImport(python, module string) bool {
	cmd := exec.Command(python, "-c", "import "+module)
	return cmd.Run() == nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// newTeeWriter 返回同时写入日志文件与控制台的 writer。
// 控制台输出按行加 prefix 前缀 (便于区分来源), 文件保持原样。
func newTeeWriter(prefix string, console io.Writer, file io.Writer) io.Writer {
	return &teeWriter{prefix: prefix, console: console, file: file}
}

type teeWriter struct {
	prefix  string
	console io.Writer
	file    io.Writer
	mu      sync.Mutex
	pending []byte
}

func (w *teeWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 文件: 原样写入
	n, err := w.file.Write(p)
	if err != nil {
		return n, err
	}
	// 控制台: 按行加前缀 (缓冲不完整行, 避免跨 Write 断行)
	w.pending = append(w.pending, p...)
	for {
		idx := bytes.IndexByte(w.pending, '\n')
		if idx < 0 {
			break
		}
		line := w.pending[:idx+1]
		w.pending = w.pending[idx+1:]
		// 忽略控制台写入错误/空: 后台运行时 os.Stdout 可能无效, 不影响文件日志
		if w.console != nil {
			_, _ = w.console.Write([]byte(w.prefix))
			_, _ = w.console.Write(line)
		}
	}
	return n, nil
}

// runCommandEnv 运行命令并记录输出到 LotsACG 日志。
func runCommandEnv(name, dir string, env []string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		log.Error("kmua: command failed", "cmd", name, "args", strings.Join(args, " "), "err", err, "output", buf.String())
		return err
	}
	if buf.Len() > 0 {
		log.Info("kmua: command output", "cmd", name, "output", buf.String())
	}
	return nil
}

// runtimeProxy 返回用于 uv 等命令的代理地址。
func runtimeProxy() string {
	cfg := runtimecfg.Get()
	if cfg.HttpClient.Proxy != "" {
		return cfg.HttpClient.Proxy
	}
	if cfg.Source.Proxy != "" {
		return cfg.Source.Proxy
	}
	if cfg.Telegram.Proxy != "" {
		return cfg.Telegram.Proxy
	}
	return ""
}

var _ = time.Second // keep import if unused later
