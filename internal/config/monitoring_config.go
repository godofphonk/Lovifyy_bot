package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// MonitoringConfig конфигурация мониторинга
type MonitoringConfig struct {
	Enabled           bool          `json:"enabled"`
	MetricsInterval   time.Duration `json:"metrics_interval"`
	HealthCheckPort   int           `json:"health_check_port"`
	LogLevel          string        `json:"log_level"`
	EnableProfiling   bool          `json:"enable_profiling"`
	ProfilingPort     int           `json:"profiling_port"`
	AlertsEnabled     bool          `json:"alerts_enabled"`
	AlertWebhookURL   string        `json:"alert_webhook_url,omitempty"`
}

// LoadMonitoringConfig загружает конфигурацию мониторинга из переменных окружения
func LoadMonitoringConfig() MonitoringConfig {
	config := MonitoringConfig{
		Enabled:         true,                // значение по умолчанию
		MetricsInterval: 1 * time.Minute,    // значение по умолчанию
		HealthCheckPort: 8080,               // значение по умолчанию
		LogLevel:        "info",             // значение по умолчанию
		EnableProfiling: false,              // значение по умолчанию
		ProfilingPort:   6060,               // значение по умолчанию
		AlertsEnabled:   false,              // значение по умолчанию
	}

	// Загружаем включение мониторинга
	if enabledStr := os.Getenv("MONITORING_ENABLED"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			config.Enabled = enabled
		}
	}

	// Загружаем интервал метрик
	if intervalStr := os.Getenv("MONITORING_METRICS_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			config.MetricsInterval = interval
		}
	}

	// Загружаем порт health check
	if portStr := os.Getenv("MONITORING_HEALTH_CHECK_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port < 65536 {
			config.HealthCheckPort = port
		}
	}

	// Загружаем уровень логирования
	if logLevel := os.Getenv("MONITORING_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// Загружаем включение профилирования
	if profilingStr := os.Getenv("MONITORING_ENABLE_PROFILING"); profilingStr != "" {
		if profiling, err := strconv.ParseBool(profilingStr); err == nil {
			config.EnableProfiling = profiling
		}
	}

	// Загружаем порт профилирования
	if profilingPortStr := os.Getenv("MONITORING_PROFILING_PORT"); profilingPortStr != "" {
		if port, err := strconv.Atoi(profilingPortStr); err == nil && port > 0 && port < 65536 {
			config.ProfilingPort = port
		}
	}

	// Загружаем включение алертов
	if alertsStr := os.Getenv("MONITORING_ALERTS_ENABLED"); alertsStr != "" {
		if alerts, err := strconv.ParseBool(alertsStr); err == nil {
			config.AlertsEnabled = alerts
		}
	}

	// Загружаем webhook URL для алертов
	if webhookURL := os.Getenv("MONITORING_ALERT_WEBHOOK_URL"); webhookURL != "" {
		config.AlertWebhookURL = webhookURL
	}

	return config
}

// ValidateMonitoringConfig проверяет корректность конфигурации мониторинга
func (mc MonitoringConfig) Validate() error {
	if mc.MetricsInterval <= 0 {
		return fmt.Errorf("metrics interval must be positive")
	}
	
	if mc.HealthCheckPort <= 0 || mc.HealthCheckPort >= 65536 {
		return fmt.Errorf("health check port must be between 1 and 65535")
	}
	
	if mc.EnableProfiling && (mc.ProfilingPort <= 0 || mc.ProfilingPort >= 65536) {
		return fmt.Errorf("profiling port must be between 1 and 65535")
	}
	
	// Проверяем уровень логирования
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLevel := false
	for _, level := range validLogLevels {
		if mc.LogLevel == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error, fatal)", mc.LogLevel)
	}
	
	// Проверяем конфликт портов
	if mc.EnableProfiling && mc.HealthCheckPort == mc.ProfilingPort {
		return fmt.Errorf("health check port and profiling port cannot be the same")
	}
	
	return nil
}

// GetValidLogLevels возвращает список допустимых уровней логирования
func GetValidLogLevels() []string {
	return []string{"debug", "info", "warn", "error", "fatal"}
}

// IsValidLogLevel проверяет, является ли уровень логирования допустимым
func IsValidLogLevel(level string) bool {
	validLevels := GetValidLogLevels()
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}
