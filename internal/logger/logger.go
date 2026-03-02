package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger writes timestamped output for a single job.
type Logger struct {
	writer io.WriteCloser
	name   string
	path   string
	mu     sync.Mutex
}

// Write prepends a timestamp to each line written.
func (l *Logger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, string(p))
	_, err := l.writer.Write([]byte(line))
	return len(p), err
}

// Close closes the underlying writer.
func (l *Logger) Close() error {
	return l.writer.Close()
}

// Tail returns the last n lines from the log file.
func (l *Logger) Tail(n int) ([]string, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, scanner.Err()
}

// Manager creates and caches per-job loggers.
type Manager struct {
	logDir   string
	maxSize  int // MB
	maxFiles int
	mu       sync.Mutex
	loggers  map[string]*Logger
}

// NewManager creates a log manager. It ensures the log directory exists.
func NewManager(logDir string, maxSize, maxFiles int) *Manager {
	_ = os.MkdirAll(logDir, 0o755)
	return &Manager{
		logDir:   logDir,
		maxSize:  maxSize,
		maxFiles: maxFiles,
		loggers:  make(map[string]*Logger),
	}
}

// GetLogger returns (or creates) a logger for the named job.
func (m *Manager) GetLogger(jobName string) *Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	if l, ok := m.loggers[jobName]; ok {
		return l
	}

	logPath := filepath.Join(m.logDir, jobName+".log")
	w := newRotatingWriter(logPath, m.maxSize, m.maxFiles)
	l := &Logger{writer: w, name: jobName, path: logPath}
	m.loggers[jobName] = l
	return l
}

// Close closes all managed loggers.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, l := range m.loggers {
		l.Close()
	}
}
