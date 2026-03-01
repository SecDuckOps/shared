package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// createRotator simplifies setting up Lumberjack rotators for different log files.
func createRotator(logDir string, filename string) (*lumberjack.Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(logDir, filename),
		MaxSize:    50, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	}, nil
}
