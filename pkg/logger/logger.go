package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
)

type Logger struct {
	ctx      context.Context
	logFile  *os.File
	logger   *log.Logger
	logLevel LogLevel
}

func NewLogger(appDir string) *Logger {
	// Create logs directory
	logDir := filepath.Join(appDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return &Logger{
			logger:   log.New(os.Stdout, "", log.LstdFlags),
			logLevel: LevelInfo,
		}
	}

	// Open log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("app-%s.log", dateStr))
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return &Logger{
			logger:   log.New(os.Stdout, "", log.LstdFlags),
			logLevel: LevelInfo,
		}
	}

	// Multiwriter: stdout + file
	mw := io.MultiWriter(os.Stdout, file)

	return &Logger{
		logFile:  file,
		logger:   log.New(mw, "", log.LstdFlags),
		logLevel: LevelInfo,
	}
}

func (l *Logger) SetContext(ctx context.Context) {
	l.ctx = ctx
}

func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	if l.logLevel > LevelInfo {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[INFO] %s", msg)
}

func (l *Logger) Error(format string, args ...interface{}) {
	if l.logLevel > LevelError {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[ERROR] %s", msg)
	
	// Emit event to frontend for toast notifications
	if l.ctx != nil {
		wailsRuntime.EventsEmit(l.ctx, "app-error", msg)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.logLevel > LevelDebug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[DEBUG] %s", msg)
}
