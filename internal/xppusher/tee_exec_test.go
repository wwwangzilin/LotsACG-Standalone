package xppusher

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestTeeExecPython(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "out.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	cmd := exec.Command("python", "-c", "import sys; print('hello stdout'); sys.stderr.write('hello stderr\\n')")
	tee := newTeeWriter("[t] ", os.Stdout, logFile)
	cmd.Stdout = tee
	cmd.Stderr = tee
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008}
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd run err: %v", err)
	}
	logFile.Sync()
	data, _ := os.ReadFile(logPath)
	t.Logf("out.log = %q", data)
	got := strings.ReplaceAll(string(data), "\r\n", "\n")
	if got != "hello stdout\nhello stderr\n" {
		t.Fatalf("tee file content mismatch: %q", data)
	}
}
