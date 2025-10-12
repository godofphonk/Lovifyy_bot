package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"Lovifyy_bot/internal/logger"
)

// Config основная конфигурация приложения
type Config struct {
	// Telegram настройки
	Telegram TelegramConfig `json:"telegram"`
	
	// OpenAI настройки
	OpenAI OpenAIConfig `json:"openai"`
	
	// Логирование
	Logger logger.Config `json:"logger"`
	
	// База данных (файлы)
	Database DatabaseConfig `json:"database"`
	
	// Сервер настройки
	Server ServerConfig `json:"server"`
	
	// Безопасность
	Security SecurityConfig `json:"security"`
	
	// Мониторинг
	Monitoring MonitoringConfig `json:"monitoring"`
}

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	BotToken     string  `json:"bot_token"`
	AdminIDs     []int64 `json:"admin_ids"`
	SystemPrompt string  `json:"system_prompt"`
	Timeout      int     `json:"timeout"`      // секунды
	RetryCount   int     `json:"retry_count"`
}

// OpenAIConfig конфигурация OpenAI API
type OpenAIConfig struct {
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Timeout     int     `json:"timeout"`      // секунды
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	DataDir         string `json:"data_dir"`         // Корневая папка данных
	ChatsDir        string `json:"chats_dir"`        // История чатов
	DiariesDir      string `json:"diaries_dir"`      // Записи дневников
	ExercisesDir    string `json:"exercises_dir"`    // Упражнения
	LogsDir         string `json:"logs_dir"`         // Логи
	NotificationsDir string `json:"notifications_dir"` // Уведомления
	BackupEnabled   bool   `json:"backup_enabled"`
	BackupDir       string `json:"backup_dir"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsPort     int           `json:"metrics_port"`
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	RateLimitDuration time.Duration `json:"rate_limit_duration"`
	MaxMessageLength  int           `json:"max_message_length"`
	EnableSanitization bool         `json:"enable_sanitization"`
	AllowedCommands   []string      `json:"allowed_commands"`
}

// MonitoringConfig конфигурация мониторинга
type MonitoringConfig struct {
	Enabled           bool          `json:"enabled"`
	MetricsInterval   time.Duration `json:"metrics_interval"`
	HealthCheckPort   int           `json:"health_check_port"`
	AlertWebhookURL   string        `json:"alert_webhook_url"`
	EnablePrometheus  bool          `json:"enable_prometheus"`
}

// LoadConfig загружает конфигурацию из переменных окружения и файла
func LoadConfig() (*Config, error) {
	config := getDefaultConfig()
	
	// Загружаем из файла конфигурации если существует
	if err := loadFromFile(config, "config.json"); err != nil {
		// Файл не обязателен, продолжаем с переменными окружения
	}
	
	// Переопределяем переменными окружения
	loadFromEnv(config)
	
	// Валидируем конфигурацию
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return config, nil
}

// getDefaultConfig возвращает конфигурацию по умолчанию
func getDefaultConfig() *Config {
	return &Config{
		Telegram: TelegramConfig{
			SystemPrompt: "Ты - профессиональный консультант по отношениям и семейный психолог с многолетним опытом работы.",
			Timeout:      30,
			RetryCount:   3,
		},
		OpenAI: OpenAIConfig{
			Model:       "gpt-4o-mini",
			BaseURL:     "https://api.openai.com/v1",
			MaxTokens:   1500,
			Temperature: 0.7,
			Timeout:     30,
		},
		Logger: logger.GetDefaultConfig(),
		Database: DatabaseConfig{
			DataDir:         "data",
			ChatsDir:        "data/chats",
			DiariesDir:      "data/diaries",
			ExercisesDir:    "exercises",
			LogsDir:         "data/logs",
			NotificationsDir: "data/notifications",
			BackupEnabled:   true,
			BackupDir:       "data/backups",
		},
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			EnableMetrics:   true,
			MetricsPort:     9090,
		},
		Security: SecurityConfig{
			RateLimitDuration:  3 * time.Second,
			MaxMessageLength:   4000,
			EnableSanitization: true,
			AllowedCommands:    []string{"start", "help", "menu", "admin", "notify", "setweek"},
		},
		Monitoring: MonitoringConfig{
			Enabled:          true,
			MetricsInterval:  30 * time.Second,
			HealthCheckPort:  8081,
			EnablePrometheus: false,
		},
	}
}

// loadFromFile загружает конфигурацию из JSON файла
func loadFromFile(config *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, config)
}

// loadFromEnv загружает конфигурацию из переменных окружения
func loadFromEnv(config *Config) {
	// Telegram
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		config.Telegram.BotToken = token
	}
	
	if adminIDs := os.Getenv("ADMIN_IDS"); adminIDs != "" {
		config.Telegram.AdminIDs = parseAdminIDs(adminIDs)
	}
	
	if prompt := os.Getenv("SYSTEM_PROMPT"); prompt != "" {
		config.Telegram.SystemPrompt = prompt
	}
	
	// OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}
	
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		config.OpenAI.Model = model
	}
	
	// Logger
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logger.Level = level
	}
	
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logger.Format = format
	}
	
	// Server
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	
	// Monitoring
	if enablePrometheus := os.Getenv("ENABLE_PROMETHEUS"); enablePrometheus != "" {
		if enable, err := strconv.ParseBool(enablePrometheus); err == nil {
			config.Monitoring.EnablePrometheus = enable
		}
	}
}

// parseAdminIDs парсит список ID администраторов
func parseAdminIDs(adminIDsStr string) []int64 {
	var adminIDs []int64
	for _, idStr := range strings.Split(adminIDsStr, ",") {
		idStr = strings.TrimSpace(idStr)
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			adminIDs = append(adminIDs, id)
		}
	}
	return adminIDs
}

// validateConfig валидирует конфигурацию
func validateConfig(config *Config) error {
	if config.Telegram.BotToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}
	
	if len(config.Telegram.AdminIDs) == 0 {
		return fmt.Errorf("at least one admin ID is required")
	}
	
	if config.OpenAI.APIKey == "" {
		return fmt.Errorf("openai api key is required")
	}
	
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	return nil
}

// SaveConfig сохраняет конфигурацию в файл
func (c *Config) SaveConfig(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// GetTelegramConfig возвращает конфигурацию Telegram
func (c *Config) GetTelegramConfig() TelegramConfig {
	return c.Telegram
}

// GetOpenAIConfig возвращает конфигурацию OpenAI
func (c *Config) GetOpenAIConfig() OpenAIConfig {
	return c.OpenAI
}

// IsProduction проверяет, запущено ли приложение в production режиме
func (c *Config) IsProduction() bool {
	return strings.ToLower(os.Getenv("GO_ENV")) == "production"
}

// IsDevelopment проверяет, запущено ли приложение в development режиме
func (c *Config) IsDevelopment() bool {
	return !c.IsProduction()
}
