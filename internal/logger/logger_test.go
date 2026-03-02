package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerGetLogger(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir, 10, 5)
	defer mgr.Close()

	l := mgr.GetLogger("test-job")
	if l == nil {
		t.Fatal("expected non-nil logger")
	}

	// Same logger returned on second call.
	l2 := mgr.GetLogger("test-job")
	if l != l2 {
		t.Error("expected same logger instance")
	}
}

func TestLoggerWrite(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir, 10, 5)
	defer mgr.Close()

	l := mgr.GetLogger("write-test")
	n, err := l.Write([]byte("hello world\n"))
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != len("hello world\n") {
		t.Errorf("expected n=%d, got %d", len("hello world\n"), n)
	}

	// Check the file was created and contains timestamped output.
	logPath := filepath.Join(dir, "write-test.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "hello world") {
		t.Errorf("expected log content to contain 'hello world', got %q", content)
	}
	if !strings.Contains(content, "[") {
		t.Error("expected timestamp prefix in log output")
	}
}

func TestLoggerTail(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir, 10, 5)
	defer mgr.Close()

	l := mgr.GetLogger("tail-test")
	for i := 0; i < 10; i++ {
		l.Write([]byte("line\n"))
	}

	lines, err := l.Tail(3)
	if err != nil {
		t.Fatalf("Tail() error: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestLoggerTailNoFile(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{path: filepath.Join(dir, "nonexistent.log"), name: "nope"}
	lines, err := l.Tail(10)
	if err != nil {
		t.Fatalf("Tail() error: %v", err)
	}
	if lines != nil {
		t.Errorf("expected nil lines for nonexistent file, got %v", lines)
	}
}
