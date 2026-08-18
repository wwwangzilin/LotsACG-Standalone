package xppusher

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestTeeWriterFileWrite(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	// console 用一个 buffer (模拟控制台)
	var console bytes.Buffer
	w := newTeeWriter("[x] ", &console, logFile)
	if _, err := w.Write([]byte("hello\nworld\n")); err != nil {
		t.Fatalf("write err: %v", err)
	}
	logFile.Sync()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\nworld\n" {
		t.Fatalf("file content = %q, want %q", data, "hello\nworld\n")
	}
	if console.String() != "[x] hello\n[x] world\n" {
		t.Fatalf("console = %q", console.String())
	}
}

func TestTeeWriterConsoleErrorIgnored(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	// console 写失败 (nil writer)
	w := newTeeWriter("[x] ", nil, logFile)
	if _, err := w.Write([]byte("line1\nline2\n")); err != nil {
		t.Fatalf("write err should be nil even if console fails: %v", err)
	}
	logFile.Sync()
	data, _ := os.ReadFile(logPath)
	if string(data) != "line1\nline2\n" {
		t.Fatalf("file content = %q", data)
	}
}
