package config

import (
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

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	DataDir         string `json:"data_dir"`         // Корневая папка данных
	ChatsDir        string `json:"chats_dir"`        // История чатов
	DiariesDir      string `json:"diaries_dir"`      // Записи дневников
	ExercisesDir    string `json:"exercises_dir"`    // Упражнения
	NotificationsDir string `json:"notifications_dir"` // Уведомления
	BackupEnabled   bool   `json:"backup_enabled"`   // Включить резервное копирование
	BackupInterval  string `json:"backup_interval"`  // Интервал резервного копирования
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	MaxHeaderBytes  int           `json:"max_header_bytes"`
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	RateLimitDuration time.Duration `json:"rate_limit_duration"`
	MaxMessageLength  int           `json:"max_message_length"`
	EnableSanitization bool         `json:"enable_sanitization"`
	AllowedCommands   []string      `json:"allowed_commands"`
}
