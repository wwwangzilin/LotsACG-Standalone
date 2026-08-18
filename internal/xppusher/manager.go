package xppusher

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

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/kvstor"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const pidKey = "xppusher:pid"

// xppusherCodeVersion 内嵌 XP-Pusher 源码版本标记。
// 修改 internal/xppusher/project 下的源码时需同步递增, 触发重新提取覆盖。
const xppusherCodeVersion = "20260816.1"

// Manager 管理内嵌/外部部署的 Pixiv-XP-Pusher (Python) 进程。
// 日志统一写入 <exe>/logs/xppusher.log, 与 LotsACG 日志同目录。
type Manager struct {
	mu     sync.Mutex
	cfg    runtimecfg.XPPusherConfig
	exeDir string
}

func NewManager(cfg runtimecfg.XPPusherConfig) (*Manager, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, exeDir: filepath.Dir(exe)}, nil
}

// RunDir 返回 XP-Pusher 实际运行目录: 显式配置 dir 优先, 否则为 <exeDir>/xppusher (内嵌提取)。
func (m *Manager) RunDir() string {
	if m.cfg.Dir != "" {
		return m.cfg.Dir
	}
	return filepath.Join(m.exeDir, "xppusher")
}

// LogPath 返回 XP-Pusher 日志文件路径 (与 LotsACG 日志同目录)。
func (m *Manager) LogPath() string {
	return filepath.Join(m.exeDir, "logs", "xppusher.log")
}

// EnsureExtracted 确保 XP-Pusher 源码就绪:
// 显式 dir -> 校验 main.py 存在; 否则把内嵌源码提取到 <exeDir>/xppusher。
// 版本感知: 内嵌源码版本变化时重新覆盖源码, 但保留 config.yaml / data / logs。
func (m *Manager) EnsureExtracted() error {
	if m.cfg.Dir != "" {
		if _, err := os.Stat(filepath.Join(m.cfg.Dir, "main.py")); err != nil {
			return fmt.Errorf("xppusher 目录中没有 main.py: %s", m.cfg.Dir)
		}
		return nil
	}
	dir := m.RunDir()
	marker := filepath.Join(dir, ".xppusher_version")
	needExtract := false
	if _, err := os.Stat(filepath.Join(dir, "main.py")); err != nil {
		needExtract = true
	} else if data, err := os.ReadFile(marker); err != nil || strings.TrimSpace(string(data)) != xppusherCodeVersion {
		// 已提取但源码版本过旧: 重新覆盖源码 (config.yaml/data/logs 不受影响)
		needExtract = true
		log.Info("xppusher: embedded source updated, re-extracting", "dir", dir)
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
	// 初始化 config.yaml: 仅当不存在时 (升级时保留用户已有配置)
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); os.IsNotExist(err) {
		src := m.cfg.Config
		if src == "" {
			src = filepath.Join(dir, "config.example.yaml")
		}
		if data, err := os.ReadFile(src); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644)
		}
	}
	// 跳过交互式初始化向导 + 记录源码版本
	_ = os.WriteFile(filepath.Join(dir, ".initialized"), []byte(""), 0644)
	_ = os.WriteFile(marker, []byte(xppusherCodeVersion), 0644)
	log.Info("xppusher: embedded source extracted", "dir", dir)
	return nil
}

// pythonHasDeps 检查 Python 是否能满足 XP-Pusher 的关键依赖。
// 任一缺失则视为不可用, 由调用方回退到创建 venv + pip install。
func pythonHasDeps(python string) bool {
	for _, mod := range []string{"telegram", "pixivpy_async", "apscheduler"} {
		if !pythonCanImport(python, mod) {
			return false
		}
	}
	return true
}

// ResolvePython 返回可用的 Python 解释器 (显式配置 > venv > 系统)。
// 若依赖缺失, 自动创建 venv 并安装 requirements.txt。
func (m *Manager) ResolvePython(ctx context.Context, progress func(string)) (string, error) {
	if m.cfg.Python != "" {
		if pythonHasDeps(m.cfg.Python) {
			return m.cfg.Python, nil
		}
		return "", fmt.Errorf("配置的 python 依赖不完整 (需要 telegram/pixivpy_async/apscheduler): %s", m.cfg.Python)
	}
	dir := m.RunDir()
	candidates := make([]string, 0, 3)
	if runtime.GOOS == "windows" {
		candidates = append(candidates, filepath.Join(dir, ".venv", "Scripts", "python.exe"))
	} else {
		candidates = append(candidates, filepath.Join(dir, ".venv", "bin", "python"))
	}
	candidates = append(candidates, "python", "python3")
	for _, c := range candidates {
		if c != "python" && c != "python3" {
			if _, err := os.Stat(c); err != nil {
				continue
			}
		}
		if pythonHasDeps(c) {
			return c, nil
		}
	}

	// 需要创建 venv + 安装依赖
	base := "python"
	if !commandExists(base) {
		base = "python3"
	}
	if !commandExists(base) {
		return "", errors.New("未找到可用的 Python (需 3.10+), 请安装或配置 [xppusher] python")
	}
	venvDir := filepath.Join(dir, ".venv")
	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		if progress != nil {
			progress("未检测到 XP-Pusher 运行环境, 正在创建虚拟环境...")
		}
		if err := runCommand(base, dir, "-m", "venv", venvDir); err != nil {
			return "", fmt.Errorf("创建 venv 失败: %w", err)
		}
	}
	venvPy := filepath.Join(venvDir, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPy = filepath.Join(venvDir, "bin", "python")
	}
	if !pythonCanImport(venvPy, "telegram") {
		if progress != nil {
			progress("正在安装 XP-Pusher 依赖 (首次需要几分钟)...")
		}
		args := []string{"-m", "pip", "install", "-r", "requirements.txt"}
		if proxy := runtimeProxy(); proxy != "" {
			args = append(args, "--proxy", proxy)
		}
		if err := runCommand(venvPy, dir, args...); err != nil {
			return "", fmt.Errorf("安装 XP-Pusher 依赖失败: %w", err)
		}
	}
	return venvPy, nil
}

// Start 启动 XP-Pusher 进程 (日志写入 logs/xppusher.log)。
func (m *Manager) Start(ctx context.Context, progress func(string)) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pid := m.RunningPID(ctx); pid != 0 {
		return pid, nil
	}
	if err := m.EnsureExtracted(); err != nil {
		return 0, err
	}
	dir := m.RunDir()
	python, err := m.ResolvePython(ctx, progress)
	if err != nil {
		return 0, err
	}
	command := m.cfg.Command
	if command == "" {
		command = "main.py"
	}
	args := []string{command}
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
	// 日志同时写入文件 (供 /xppusher log) 与 LotsACG 控制台 (实时滚动, 带 [xppusher] 前缀)
	execCmd.Stdout = newTeeWriter("[xppusher] ", os.Stdout, logFile)
	execCmd.Stderr = newTeeWriter("[xppusher] ", os.Stderr, logFile)
	// 强制 Python 以 UTF-8 + 无缓冲输出: 避免 Windows 默认 GBK 编码乱码, 以及管道重定向下
	// stdout/stderr 块缓冲导致日志不实时写入 (tee 到控制台/日志文件)
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
	log.Info("xppusher: started", "pid", execCmd.Process.Pid, "dir", dir, "python", python, "log", m.LogPath())
	return execCmd.Process.Pid, nil
}

// Stop 停止 XP-Pusher 进程。
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
	log.Info("xppusher: stopped", "pid", pid)
	return pid, nil
}

// RunningPID 返回正在运行的 XP-Pusher 进程 PID, 0 表示未运行。
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

// LogTail 返回 XP-Pusher 日志最后 n 行。
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
// 处理历史 GBK 日志与新 UTF-8 日志混合的情况。
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

// runCommand 运行命令并记录输出到 LotsACG 日志。
func runCommand(name, dir string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		log.Error("xppusher: command failed", "cmd", name, "args", strings.Join(args, " "), "err", err, "output", buf.String())
		return err
	}
	if buf.Len() > 0 {
		log.Info("xppusher: command output", "cmd", name, "output", buf.String())
	}
	return nil
}

// runtimeProxy 返回用于 pip 等命令的代理地址。
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
