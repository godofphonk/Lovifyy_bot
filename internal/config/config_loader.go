package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/logger"
)

// Load загружает конфигурацию из переменных окружения и файла
func Load() (*Config, error) {
	config := &Config{}

	// Загружаем конфигурацию Telegram
	telegramConfig, err := LoadTelegramConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Telegram config: %w", err)
	}
	config.Telegram = telegramConfig

	// Загружаем конфигурацию OpenAI
	openAIConfig, err := LoadOpenAIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAI config: %w", err)
	}
	config.OpenAI = openAIConfig

	// Загружаем конфигурацию мониторинга
	config.Monitoring = LoadMonitoringConfig()

	// Загружаем остальные конфигурации
	config.Logger = loadLoggerConfig()
	config.Database = loadDatabaseConfig()
	config.Server = loadServerConfig()
	config.Security = loadSecurityConfig()

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// LoadFromFile загружает конфигурацию из JSON файла
func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// SaveToFile сохраняет конфигурацию в JSON файл
func (c *Config) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate проверяет корректность всей конфигурации
func (c *Config) Validate() error {
	if err := c.Telegram.Validate(); err != nil {
		return fmt.Errorf("telegram config: %w", err)
	}

	if err := c.OpenAI.Validate(); err != nil {
		return fmt.Errorf("openai config: %w", err)
	}

	if err := c.Monitoring.Validate(); err != nil {
		return fmt.Errorf("monitoring config: %w", err)
	}

	return nil
}

// loadLoggerConfig загружает конфигурацию логгера
func loadLoggerConfig() logger.Config {
	config := logger.Config{
		Level:  "info",  // значение по умолчанию
		Format: "json",  // значение по умолчанию
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = level
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = format
	}

	return config
}

// loadDatabaseConfig загружает конфигурацию базы данных
func loadDatabaseConfig() DatabaseConfig {
	config := DatabaseConfig{
		DataDir:          "data",           // значение по умолчанию
		ChatsDir:         "data/chats",     // значение по умолчанию
		DiariesDir:       "data/diaries",   // значение по умолчанию
		ExercisesDir:     "data/exercises", // значение по умолчанию
		NotificationsDir: "data/notifications", // значение по умолчанию
		BackupEnabled:    false,            // значение по умолчанию
		BackupInterval:   "24h",            // значение по умолчанию
	}

	if dataDir := os.Getenv("DATABASE_DATA_DIR"); dataDir != "" {
		config.DataDir = dataDir
	}

	if chatsDir := os.Getenv("DATABASE_CHATS_DIR"); chatsDir != "" {
		config.ChatsDir = chatsDir
	}

	if diariesDir := os.Getenv("DATABASE_DIARIES_DIR"); diariesDir != "" {
		config.DiariesDir = diariesDir
	}

	if exercisesDir := os.Getenv("DATABASE_EXERCISES_DIR"); exercisesDir != "" {
		config.ExercisesDir = exercisesDir
	}

	if notificationsDir := os.Getenv("DATABASE_NOTIFICATIONS_DIR"); notificationsDir != "" {
		config.NotificationsDir = notificationsDir
	}

	if backupStr := os.Getenv("DATABASE_BACKUP_ENABLED"); backupStr != "" {
		if backup, err := strconv.ParseBool(backupStr); err == nil {
			config.BackupEnabled = backup
		}
	}

	if interval := os.Getenv("DATABASE_BACKUP_INTERVAL"); interval != "" {
		config.BackupInterval = interval
	}

	return config
}

// loadServerConfig загружает конфигурацию сервера
func loadServerConfig() ServerConfig {
	config := ServerConfig{
		Port:           8080,             // значение по умолчанию
		ReadTimeout:    30 * time.Second, // значение по умолчанию
		WriteTimeout:   30 * time.Second, // значение по умолчанию
		IdleTimeout:    60 * time.Second, // значение по умолчанию
		MaxHeaderBytes: 1 << 20,          // 1MB по умолчанию
	}

	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port < 65536 {
			config.Port = port
		}
	}

	if timeoutStr := os.Getenv("SERVER_READ_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.ReadTimeout = timeout
		}
	}

	if timeoutStr := os.Getenv("SERVER_WRITE_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.WriteTimeout = timeout
		}
	}

	if timeoutStr := os.Getenv("SERVER_IDLE_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.IdleTimeout = timeout
		}
	}

	if bytesStr := os.Getenv("SERVER_MAX_HEADER_BYTES"); bytesStr != "" {
		if bytes, err := strconv.Atoi(bytesStr); err == nil && bytes > 0 {
			config.MaxHeaderBytes = bytes
		}
	}

	return config
}

// loadSecurityConfig загружает конфигурацию безопасности
func loadSecurityConfig() SecurityConfig {
	config := SecurityConfig{
		RateLimitDuration:  1 * time.Minute, // значение по умолчанию
		MaxMessageLength:   4000,             // значение по умолчанию
		EnableSanitization: true,             // значение по умолчанию
		AllowedCommands:    []string{},       // значение по умолчанию
	}

	if durationStr := os.Getenv("SECURITY_RATE_LIMIT_DURATION"); durationStr != "" {
		if duration, err := time.ParseDuration(durationStr); err == nil {
			config.RateLimitDuration = duration
		}
	}

	if lengthStr := os.Getenv("SECURITY_MAX_MESSAGE_LENGTH"); lengthStr != "" {
		if length, err := strconv.Atoi(lengthStr); err == nil && length > 0 {
			config.MaxMessageLength = length
		}
	}

	if sanitizeStr := os.Getenv("SECURITY_ENABLE_SANITIZATION"); sanitizeStr != "" {
		if sanitize, err := strconv.ParseBool(sanitizeStr); err == nil {
			config.EnableSanitization = sanitize
		}
	}

	return config
}
