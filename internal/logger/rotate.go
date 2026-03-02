package logger

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

// newRotatingWriter creates a lumberjack-based rotating log writer.
func newRotatingWriter(path string, maxSizeMB, maxFiles int) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSizeMB,
		MaxBackups: maxFiles,
		Compress:   false,
	}
}
