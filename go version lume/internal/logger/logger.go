package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu      sync.RWMutex
	logFile *os.File
	logger  *log.Logger
)

func rotateLog(path string) {
	if info, err := os.Stat(path); err == nil && info.Size() > 5*1024*1024 {
		oldPath := path + ".old"
		os.Remove(oldPath)
		os.Rename(path, oldPath)
	}
}

func Init() error {
	exePath, err := os.Executable()
	var logPath string

	if err == nil {
		dir := filepath.Dir(exePath)

		testPath := filepath.Join(dir, "lume_app.log")
		rotateLog(testPath)
		f, err := os.OpenFile(testPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			logFile = f
			logPath = testPath
		}
	}

	if logFile == nil {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("USERPROFILE")
		}
		lumeDir := filepath.Join(appData, "Lume")
		os.MkdirAll(lumeDir, 0755)
		logPath = filepath.Join(lumeDir, "lume_app.log")
		rotateLog(logPath)
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		logFile = f
	}

	mu.Lock()
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Println("--- Lume Started ---")
	mu.Unlock()
	return nil
}

func sanitizeLogMessage(format string, v ...interface{}) string {
	msg := fmt.Sprintf(format, v...)
	msg = strings.ReplaceAll(msg, "\r", "\\r")
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

func Info(format string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if logger != nil {
		logger.Printf("[INFO] %s", sanitizeLogMessage(format, v...))
	}
}

func Error(format string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if logger != nil {
		logger.Printf("[ERROR] %s", sanitizeLogMessage(format, v...))
	}
}

func Fatal(format string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if logger != nil {
		logger.Printf("[FATAL] %s", sanitizeLogMessage(format, v...))
		if logFile != nil {
			logFile.Sync()
		}
	}
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		if logger != nil {
			logger.Println("--- Lume Closed ---")
		}
		logFile.Close()
		logFile = nil
		logger = nil
	}
}
