package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Logger
	once     sync.Once
	logDir   string
)

type Logger struct {
	mu          sync.Mutex
	currentDate string
	currentFile *os.File
}

func InitLogger(dir string) error {
	logDir = dir
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	once.Do(func() {
		instance = &Logger{}
	})

	if err := instance.rotate(); err != nil {
		return err
	}

	go instance.startCleanupRoutine()

	return nil
}

func GetLogger() *Logger {
	if instance == nil {
		once.Do(func() {
			instance = &Logger{}
		})
	}
	return instance
}

func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	today := time.Now().Format("2006-01-02")

	if l.currentDate == today && l.currentFile != nil {
		return nil
	}

	if l.currentFile != nil {
		l.currentFile.Close()
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("textminer_%s.log", today))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.currentDate = today
	l.currentFile = file

	return nil
}

func (l *Logger) startCleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanupOldLogs()
	}
}

func (l *Logger) cleanupOldLogs() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}

	cutoffDate := time.Now().AddDate(0, 0, -30)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < 15 || name[:10] != "textminer_" || name[len(name)-4:] != ".log" {
			continue
		}

		dateStr := name[10 : len(name)-4]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoffDate) {
			os.Remove(filepath.Join(logDir, name))
		}
	}

	return nil
}

func (l *Logger) write(level, message string) error {
	if err := l.rotate(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile == nil {
		return fmt.Errorf("日志文件未初始化")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	_, err := l.currentFile.WriteString(logLine)
	return err
}

func Info(message string) {
	GetLogger().write("INFO", message)
}

func Error(message string) {
	GetLogger().write("ERROR", message)
}

func Warn(message string) {
	GetLogger().write("WARN", message)
}

func Infof(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	GetLogger().write("INFO", message)
}

func Errorf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	GetLogger().write("ERROR", message)
}

func Warnf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	GetLogger().write("WARN", message)
}
