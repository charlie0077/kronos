package logger

import (
	"bufio"
	"errors"
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
	if l.writer == nil {
		return 0, fmt.Errorf("logger %q is read-only", l.name)
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, string(p))
	_, err := l.writer.Write([]byte(line))
	return len(p), err
}

// Close closes the underlying writer.
func (l *Logger) Close() error {
	if l.writer == nil {
		return nil
	}
	return l.writer.Close()
}

// Tail returns the last n lines from the log file.
// It uses a ring buffer to avoid loading the entire file into memory.
func (l *Logger) Tail(n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	f, err := os.Open(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	ring := make([]string, n)
	total := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ring[total%n] = scanner.Text()
		total++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if total == 0 {
		return nil, nil
	}

	count := total
	if count > n {
		count = n
	}
	result := make([]string, count)
	start := total - count
	for i := 0; i < count; i++ {
		result[i] = ring[(start+i)%n]
	}
	return result, nil
}

// NewReadOnlyLogger returns a Logger with name and path set but writer left nil.
// This is used by the CLI to call Tail() without needing a full Manager.
func NewReadOnlyLogger(name, path string) *Logger {
	return &Logger{name: name, path: path}
}

// Path returns the log file path.
func (l *Logger) Path() string {
	return l.path
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
