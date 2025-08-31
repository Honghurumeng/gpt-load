package utils

import (
	"gpt-load/internal/types"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// SetupLogger configures the logging system based on the provided configuration.
func SetupLogger(configManager types.ConfigManager) {
	logConfig := configManager.GetLogConfig()

	// Set log level
	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	if logConfig.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00", // ISO 8601 format
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 检查是否启用了静默模式
	if os.Getenv("SILENT_MODE") == "true" {
		// 静默模式：只输出到文件，不输出到控制台
		if logConfig.EnableFile {
			logDir := filepath.Dir(logConfig.FilePath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				// 不输出警告日志
			} else {
				logFile, err := os.OpenFile(logConfig.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					// 不输出警告日志
				} else {
					// 只输出到文件，不输出到控制台
					logrus.SetOutput(logFile)
				}
			}
		} else {
			// 如果没有启用文件日志，则完全禁用日志输出
			logrus.SetOutput(io.Discard)
		}
	} else {
		// 正常模式：输出到控制台和文件
		if logConfig.EnableFile {
			logDir := filepath.Dir(logConfig.FilePath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				logrus.Warnf("Failed to create log directory: %v", err)
			} else {
				logFile, err := os.OpenFile(logConfig.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					logrus.Warnf("Failed to open log file: %v", err)
				} else {
					logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))
				}
			}
		}
	}
}
