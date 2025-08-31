// Package config provides configuration management for the application
package config

import (
	"fmt"
	"os"
	"strings"

	"gpt-load/internal/errors"
	"gpt-load/internal/types"
	"gpt-load/internal/utils"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Constants represents configuration constants
type Constants struct {
	MinPort               int
	MaxPort               int
	MinTimeout            int
	DefaultTimeout        int
	DefaultMaxSockets     int
	DefaultMaxFreeSockets int
}

// DefaultConstants holds default configuration values
var DefaultConstants = Constants{
	MinPort:               1,
	MaxPort:               65535,
	MinTimeout:            1,
	DefaultTimeout:        30,
	DefaultMaxSockets:     50,
	DefaultMaxFreeSockets: 10,
}

// Manager implements the ConfigManager interface
type Manager struct {
	config          *Config
	settingsManager *SystemSettingsManager
}

// Config represents the application configuration
type Config struct {
	Server      types.ServerConfig      `json:"server"`
	Auth        types.AuthConfig        `json:"auth"`
	CORS        types.CORSConfig        `json:"cors"`
	Performance types.PerformanceConfig `json:"performance"`
	Log         types.LogConfig         `json:"log"`
	Database    types.DatabaseConfig    `json:"database"`
	RedisDSN    string                  `json:"redis_dsn"`
}

// NewManager creates a new configuration manager
func NewManager(settingsManager *SystemSettingsManager) (types.ConfigManager, error) {
	manager := &Manager{
		settingsManager: settingsManager,
	}
	if err := manager.ReloadConfig(); err != nil {
		return nil, err
	}
	return manager, nil
}

// ReloadConfig reloads the configuration from environment variables
func (m *Manager) ReloadConfig() error {
	// 检查.env文件是否存在
	var envFileExists bool
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		// 保存原始的SILENT_MODE值
		originalSilentMode := os.Getenv("SILENT_MODE")
		// 设置静默模式，禁用项目日志输出
		os.Setenv("SILENT_MODE", "true")
		
		// .env文件不存在，询问用户是否创建
		fmt.Println("未找到.env文件，是否要创建一个.env文件？(y/n): ")
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			// 询问用户配置信息
			fmt.Println("请输入配置信息：")
			
			// 询问PORT
			fmt.Print("请输入端口号 (默认3001): ")
			var port string
			fmt.Scanln(&port)
			if port == "" {
				port = "3001"
			}
			
			// 询问HOST
			fmt.Print("请输入主机地址 (默认0.0.0.0): ")
			var host string
			fmt.Scanln(&host)
			if host == "" {
				host = "0.0.0.0"
			}
			
			// 询问AUTH_KEY
			fmt.Print("请输入认证密钥 (默认sk-123456): ")
			var authKey string
			fmt.Scanln(&authKey)
			if authKey == "" {
				authKey = "sk-123456"
			}
			
			// 创建.env文件内容
			defaultEnv := fmt.Sprintf(`# 服务器配置
PORT=%s
HOST=%s

# 服务器读取、写入和空闲连接的超时时间（秒）
SERVER_READ_TIMEOUT=60
SERVER_WRITE_TIMEOUT=600
SERVER_IDLE_TIMEOUT=120
SERVER_GRACEFUL_SHUTDOWN_TIMEOUT=10

# 从节点标识
IS_SLAVE=false

# 时区
TZ=Asia/Shanghai

# 认证配置 是必需的，用于保护管理 API 和 UI 界面
AUTH_KEY=%s

# 数据库配置 默认不填写，使用./data/gpt-load.db的SQLite
# MySQL 示例:
# DATABASE_DSN=root:123456@tcp(mysql:3306)/gpt-load?charset=utf8mb4&parseTime=True&loc=Local
# PostgreSQL 示例:
# DATABASE_DSN=postgres://postgres:123456@postgres:5432/gpt-load?sslmode=disable

# Redis配置 默认不填写，使用内存存储
# REDIS_DSN=redis://redis:6379/0

# 并发数量
MAX_CONCURRENT_REQUESTS=100

# CORS配置
ENABLE_CORS=true
ALLOWED_ORIGINS=*
ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
ALLOWED_HEADERS=*
ALLOW_CREDENTIALS=false

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=text
LOG_ENABLE_FILE=true
LOG_FILE_PATH=./data/logs/app.log`, port, host, authKey)
			
			// 写入.env文件
			if err := os.WriteFile(".env", []byte(defaultEnv), 0644); err != nil {
				fmt.Printf("创建.env文件失败: %v\n", err)
			} else {
				fmt.Println("已创建.env文件")
				envFileExists = true
			}
		} else {
			fmt.Println("未创建.env文件，将使用默认配置")
			envFileExists = false
		}
		
		// 恢复原始的SILENT_MODE值
		if originalSilentMode == "" {
			os.Unsetenv("SILENT_MODE")
		} else {
			os.Setenv("SILENT_MODE", originalSilentMode)
		}
	} else {
		// .env文件存在
		envFileExists = true
	}
	
	// 尝试加载.env文件
	if err := godotenv.Load(); err != nil {
		// 不显示这条日志信息
	}

	// 如果.env文件不存在或者加载失败，设置默认的环境变量
	if !envFileExists {
		// 设置默认的环境变量
		if os.Getenv("PORT") == "" {
			os.Setenv("PORT", "3001")
		}
		if os.Getenv("HOST") == "" {
			os.Setenv("HOST", "0.0.0.0")
		}
		if os.Getenv("AUTH_KEY") == "" {
			os.Setenv("AUTH_KEY", "sk-123456")
		}
	}
	config := &Config{
		Server: types.ServerConfig{
			IsMaster:                !utils.ParseBoolean(os.Getenv("IS_SLAVE"), false),
			Port:                    utils.ParseInteger(os.Getenv("PORT"), 3001),
			Host:                    utils.GetEnvOrDefault("HOST", "0.0.0.0"),
			ReadTimeout:             utils.ParseInteger(os.Getenv("SERVER_READ_TIMEOUT"), 60),
			WriteTimeout:            utils.ParseInteger(os.Getenv("SERVER_WRITE_TIMEOUT"), 600),
			IdleTimeout:             utils.ParseInteger(os.Getenv("SERVER_IDLE_TIMEOUT"), 120),
			GracefulShutdownTimeout: utils.ParseInteger(os.Getenv("SERVER_GRACEFUL_SHUTDOWN_TIMEOUT"), 10),
		},
		Auth: types.AuthConfig{
			Key: os.Getenv("AUTH_KEY"),
		},
		CORS: types.CORSConfig{
			Enabled:          utils.ParseBoolean(os.Getenv("ENABLE_CORS"), true),
			AllowedOrigins:   utils.ParseArray(os.Getenv("ALLOWED_ORIGINS"), []string{"*"}),
			AllowedMethods:   utils.ParseArray(os.Getenv("ALLOWED_METHODS"), []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders:   utils.ParseArray(os.Getenv("ALLOWED_HEADERS"), []string{"*"}),
			AllowCredentials: utils.ParseBoolean(os.Getenv("ALLOW_CREDENTIALS"), false),
		},
		Performance: types.PerformanceConfig{
			MaxConcurrentRequests: utils.ParseInteger(os.Getenv("MAX_CONCURRENT_REQUESTS"), 100),
		},
		Log: types.LogConfig{
			Level:      utils.GetEnvOrDefault("LOG_LEVEL", "info"),
			Format:     utils.GetEnvOrDefault("LOG_FORMAT", "text"),
			EnableFile: utils.ParseBoolean(os.Getenv("LOG_ENABLE_FILE"), false),
			FilePath:   utils.GetEnvOrDefault("LOG_FILE_PATH", "./data/logs/app.log"),
		},
		Database: types.DatabaseConfig{
			DSN: utils.GetEnvOrDefault("DATABASE_DSN", "./data/gpt-load.db"),
		},
		RedisDSN: os.Getenv("REDIS_DSN"),
	}
	m.config = config

	// Validate configuration
	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}

// IsMaster returns Server mode
func (m *Manager) IsMaster() bool {
	return m.config.Server.IsMaster
}

// GetAuthConfig returns authentication configuration
func (m *Manager) GetAuthConfig() types.AuthConfig {
	return m.config.Auth
}

// GetCORSConfig returns CORS configuration
func (m *Manager) GetCORSConfig() types.CORSConfig {
	return m.config.CORS
}

// GetPerformanceConfig returns performance configuration
func (m *Manager) GetPerformanceConfig() types.PerformanceConfig {
	return m.config.Performance
}

// GetLogConfig returns logging configuration
func (m *Manager) GetLogConfig() types.LogConfig {
	return m.config.Log
}

// GetRedisDSN returns the Redis DSN string.
func (m *Manager) GetRedisDSN() string {
	return m.config.RedisDSN
}

// GetDatabaseConfig returns the database configuration.
func (m *Manager) GetDatabaseConfig() types.DatabaseConfig {
	return m.config.Database
}

// GetEffectiveServerConfig returns server configuration merged with system settings
func (m *Manager) GetEffectiveServerConfig() types.ServerConfig {
	return m.config.Server
}

// Validate validates the configuration
func (m *Manager) Validate() error {
	var validationErrors []string

	// Validate port
	if m.config.Server.Port < DefaultConstants.MinPort || m.config.Server.Port > DefaultConstants.MaxPort {
		validationErrors = append(validationErrors, fmt.Sprintf("port must be between %d-%d", DefaultConstants.MinPort, DefaultConstants.MaxPort))
	}

	if m.config.Performance.MaxConcurrentRequests < 1 {
		validationErrors = append(validationErrors, "max concurrent requests cannot be less than 1")
	}

	// Validate auth key
	if m.config.Auth.Key == "" {
		validationErrors = append(validationErrors, "AUTH_KEY is required and cannot be empty")
	}

	// Validate GracefulShutdownTimeout and reset if necessary
	if m.config.Server.GracefulShutdownTimeout < 10 {
		logrus.Warnf("SERVER_GRACEFUL_SHUTDOWN_TIMEOUT value %ds is too short, resetting to minimum 10s.", m.config.Server.GracefulShutdownTimeout)
		m.config.Server.GracefulShutdownTimeout = 10
	}

	if len(validationErrors) > 0 {
		logrus.Error("Configuration validation failed:")
		for _, err := range validationErrors {
			logrus.Errorf("   - %s", err)
		}
		return errors.NewAPIError(errors.ErrValidation, strings.Join(validationErrors, "; "))
	}

	return nil
}

// DisplayServerConfig displays current server-related configuration information
func (m *Manager) DisplayServerConfig() {
	serverConfig := m.GetEffectiveServerConfig()
	corsConfig := m.GetCORSConfig()
	perfConfig := m.GetPerformanceConfig()
	logConfig := m.GetLogConfig()
	dbConfig := m.GetDatabaseConfig()

	logrus.Info("")
	logrus.Info("======= Server Configuration =======")
	logrus.Info("  --- Server ---")
	logrus.Infof("    Listen Address: %s:%d", serverConfig.Host, serverConfig.Port)
	logrus.Infof("    Graceful Shutdown Timeout: %d seconds", serverConfig.GracefulShutdownTimeout)
	logrus.Infof("    Read Timeout: %d seconds", serverConfig.ReadTimeout)
	logrus.Infof("    Write Timeout: %d seconds", serverConfig.WriteTimeout)
	logrus.Infof("    Idle Timeout: %d seconds", serverConfig.IdleTimeout)

	logrus.Info("  --- Performance ---")
	logrus.Infof("    Max Concurrent Requests: %d", perfConfig.MaxConcurrentRequests)

	logrus.Info("  --- Security ---")
	logrus.Infof("    Authentication: enabled (key loaded)")
	corsStatus := "disabled"
	if corsConfig.Enabled {
		corsStatus = fmt.Sprintf("enabled (Origins: %s)", strings.Join(corsConfig.AllowedOrigins, ", "))
	}
	logrus.Infof("    CORS: %s", corsStatus)

	logrus.Info("  --- Logging ---")
	logrus.Infof("    Log Level: %s", logConfig.Level)
	logrus.Infof("    Log Format: %s", logConfig.Format)
	logrus.Infof("    File Logging: %t", logConfig.EnableFile)
	if logConfig.EnableFile {
		logrus.Infof("    Log File Path: %s", logConfig.FilePath)
	}

	logrus.Info("  --- Dependencies ---")
	if dbConfig.DSN != "" {
		logrus.Info("    Database: configured")
	} else {
		logrus.Info("    Database: not configured")
	}
	if m.config.RedisDSN != "" {
		logrus.Info("    Redis: configured")
	} else {
		logrus.Info("    Redis: not configured")
	}
	logrus.Info("====================================")
	logrus.Info("")
}
