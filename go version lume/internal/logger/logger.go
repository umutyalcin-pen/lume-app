package logger

import (
	"log"
	"os"
	"path/filepath"
)

var (
	logFile *os.File
	logger  *log.Logger
)

// Init sets up the logger relative to the executable path.
func Init() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	
	logPath := filepath.Join(filepath.Dir(exePath), "lume_app.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	
	logFile, logger = f, log.New(f, "", log.LstdFlags)
	logger.Println("--- Lume Started ---")
	return nil
}

func Info(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[ERROR] "+format, v...)
	}
}

func Fatal(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[FATAL] "+format, v...)
		logFile.Sync()
	}
}

func Close() {
	if logFile != nil {
		logger.Println("--- Lume Closed ---")
		logFile.Close()
	}
}
