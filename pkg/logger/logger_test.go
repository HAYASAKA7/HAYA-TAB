package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	defer logger.Close()

	// Check that log file was created
	logDir := filepath.Join(tmpDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}

	// Check that log file exists
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, "app-"+dateStr+".log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogger_Info(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Log an info message
	logger.Info("Test info message: %s", "hello")

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "[INFO]") {
		t.Error("Log file should contain [INFO] tag")
	}
	if !strings.Contains(logContent, "Test info message: hello") {
		t.Error("Log file should contain the logged message")
	}
}

func TestLogger_Error(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Log an error message
	logger.Error("Test error message: %s", "error occurred")

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "[ERROR]") {
		t.Error("Log file should contain [ERROR] tag")
	}
	if !strings.Contains(logContent, "Test error message: error occurred") {
		t.Error("Log file should contain the logged message")
	}
}

func TestLogger_Debug(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Set log level to Debug to enable debug messages
	logger.logLevel = LevelDebug

	// Log a debug message
	logger.Debug("Test debug message: %d", 42)

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "[DEBUG]") {
		t.Error("Log file should contain [DEBUG] tag")
	}
	if !strings.Contains(logContent, "Test debug message: 42") {
		t.Error("Log file should contain the logged message")
	}
}

func TestLogger_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)

	// Close should not error
	logger.Close()

	// Multiple closes should be safe
	logger.Close()
}

func TestLogger_InvalidDirectory(t *testing.T) {
	// Use an invalid path that cannot be created
	invalidPath := "/invalid/path/that/does/not/exist/and/cannot/be/created"

	logger := NewLogger(invalidPath)
	if logger == nil {
		t.Fatal("NewLogger should return a logger even with invalid path")
	}
	defer logger.Close()

	// Should still be able to log (to stdout only)
	logger.Info("Test message")
}

func TestLogger_MultipleMessages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Set log level to Debug to enable all messages
	logger.logLevel = LevelDebug

	// Log multiple messages
	logger.Info("Message 1")
	logger.Error("Message 2")
	logger.Debug("Message 3")
	logger.Info("Message 4")

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Check all messages are present
	if !strings.Contains(logContent, "Message 1") {
		t.Error("Log should contain Message 1")
	}
	if !strings.Contains(logContent, "Message 2") {
		t.Error("Log should contain Message 2")
	}
	if !strings.Contains(logContent, "Message 3") {
		t.Error("Log should contain Message 3")
	}
	if !strings.Contains(logContent, "Message 4") {
		t.Error("Log should contain Message 4")
	}
}

func TestLogger_NoContextAPIRegression(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Logger no longer exposes context wiring in Wails v3;
	// basic logging should still work without any runtime context setup.
	logger.Info("Test message without context API")
}

func TestLogger_EmptyMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Log empty messages (should not panic)
	logger.Info("")
	logger.Error("")
	logger.Debug("")
}

func TestLogger_LongMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Log a very long message
	longMsg := strings.Repeat("A", 10000)
	logger.Info("%s", longMsg)

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), longMsg) {
		t.Error("Log should contain the long message")
	}
}

func TestLogger_SpecialCharacters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Log messages with special characters
	logger.Info("Message with newline\n")
	logger.Info("Message with tab\t")
	logger.Info("Message with unicode: 你好世界 🎸")

	// Read log file
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "你好世界") {
		t.Error("Log should contain unicode characters")
	}
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewLogger(tmpDir)
	defer logger.Close()

	// Write from multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				logger.Info("Goroutine %d message %d", id, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify log file exists and has content
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(tmpDir, "logs", "app-"+dateStr+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file should have content from concurrent writes")
	}
}
