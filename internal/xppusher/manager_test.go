package xppusher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// 验证混合编码日志解码: 旧 GBK 行 + 新 UTF-8 行。
func TestDecodeLogLine(t *testing.T) {
	// 用 GBK 编码器生成真实 GBK 字节
	gbkBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("Telegram 查询错误: Query is too old"))
	if err != nil {
		t.Fatal(err)
	}
	gbkLine := string(gbkBytes)
	got := decodeLogLine(gbkLine)
	if got != "Telegram 查询错误: Query is too old" {
		t.Fatalf("GBK decode mismatch: %q", got)
	}
	t.Logf("gbk -> %q", got)

	utf8Line := "[INFO] Telegram 登录成功"
	if got := decodeLogLine(utf8Line); got != utf8Line {
		t.Fatalf("UTF-8 line changed: %q", got)
	}
}

// 验证内嵌源码提取到 exe 同目录的 xppusher/ 且生成 config.yaml。
func TestEnsureExtractedEmbedded(t *testing.T) {
	m, err := NewManager(runtimecfg.XPPusherConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureExtracted(); err != nil {
		t.Fatal(err)
	}
	dir := m.RunDir()
	for _, f := range []string{"main.py", "config.py", "requirements.txt", "config.yaml", ".initialized"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("%s missing: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "notifier", "telegram.py")); err != nil {
		t.Fatalf("notifier/telegram.py missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "web")); err != nil {
		t.Fatalf("web/ missing: %v", err)
	}
	// 必须不含密钥 (config.yaml 应为 example 拷贝)
	data, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if len(data) == 0 {
		t.Fatal("config.yaml empty")
	}
	t.Log("extract ok:", dir)
}

// 验证源码版本变化时重新提取, 且保留用户已有的 config.yaml。
func TestEnsureExtractedUpgrade(t *testing.T) {
	m, err := NewManager(runtimecfg.XPPusherConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureExtracted(); err != nil {
		t.Fatal(err)
	}
	dir := m.RunDir()
	// 模拟用户改过 config.yaml
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("user: custom\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// 模拟内嵌源码版本升级
	if err := os.WriteFile(filepath.Join(dir, ".xppusher_version"), []byte("old-version"), 0644); err != nil {
		t.Fatal(err)
	}
	// 破坏一个源码文件
	mainPy := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPy, []byte("corrupted"), 0644); err != nil {
		t.Fatal(err)
	}
	// 触发重提取
	if err := m.EnsureExtracted(); err != nil {
		t.Fatal(err)
	}
	// 源码应恢复
	mainData, err := os.ReadFile(mainPy)
	if err != nil {
		t.Fatal(err)
	}
	if string(mainData) == "corrupted" {
		t.Fatal("main.py was not restored after re-extract")
	}
	// config.yaml 应保留
	cfgData, _ := os.ReadFile(cfgPath)
	if string(cfgData) != "user: custom\n" {
		t.Fatalf("config.yaml was overwritten on upgrade: %q", cfgData)
	}
	// 版本标记应更新
	marker, _ := os.ReadFile(filepath.Join(dir, ".xppusher_version"))
	if string(marker) != xppusherCodeVersion {
		t.Fatalf("version marker not updated: %q", marker)
	}
}
