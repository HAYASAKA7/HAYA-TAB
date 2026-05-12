package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

type safeMultiWriter struct {
	writers []io.Writer
}

func (t *safeMultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		w.Write(p)
	}
	return len(p), nil
}

type Logger struct {
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

	// Multiwriter: stdout + file (ignoring errors from stdout on Windows GUI)
	mw := &safeMultiWriter{writers: []io.Writer{os.Stdout, file}}

	return &Logger{
		logFile:  file,
		logger:   log.New(mw, "", log.LstdFlags),
		logLevel: LevelInfo,
	}
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

func (l *Logger) Warning(format string, args ...interface{}) {
	if l.logLevel > LevelWarning {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[WARN] %s", msg)
}

func (l *Logger) Error(format string, args ...interface{}) {
	if l.logLevel > LevelError {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[ERROR] %s", msg)

	// Emit event to frontend for toast notifications
	if app := application.Get(); app != nil {
		app.Event.Emit("app-error", msg)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.logLevel > LevelDebug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[DEBUG] %s", msg)
}
