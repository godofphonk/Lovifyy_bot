package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	BotToken     string  `json:"bot_token"`
	AdminIDs     []int64 `json:"admin_ids"`
	SystemPrompt string  `json:"system_prompt"`
	Timeout      int     `json:"timeout"`      // секунды
	RetryCount   int     `json:"retry_count"`
}

// LoadTelegramConfig загружает конфигурацию Telegram из переменных окружения
func LoadTelegramConfig() (TelegramConfig, error) {
	config := TelegramConfig{
		Timeout:    30,  // значение по умолчанию
		RetryCount: 3,   // значение по умолчанию
	}

	// Загружаем токен бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return config, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	config.BotToken = botToken

	// Загружаем список админов
	adminIDsStr := os.Getenv("TELEGRAM_ADMIN_IDS")
	if adminIDsStr != "" {
		adminIDs, err := parseAdminIDs(adminIDsStr)
		if err != nil {
			return config, fmt.Errorf("failed to parse TELEGRAM_ADMIN_IDS: %w", err)
		}
		config.AdminIDs = adminIDs
	}

	// Загружаем системный промпт
	systemPrompt := os.Getenv("TELEGRAM_SYSTEM_PROMPT")
	if systemPrompt != "" {
		config.SystemPrompt = systemPrompt
	} else {
		config.SystemPrompt = getDefaultSystemPrompt()
	}

	// Загружаем timeout
	if timeoutStr := os.Getenv("TELEGRAM_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			config.Timeout = timeout
		}
	}

	// Загружаем retry count
	if retryStr := os.Getenv("TELEGRAM_RETRY_COUNT"); retryStr != "" {
		if retry, err := strconv.Atoi(retryStr); err == nil && retry > 0 {
			config.RetryCount = retry
		}
	}

	return config, nil
}

// parseAdminIDs парсит строку с ID админов
func parseAdminIDs(adminIDsStr string) ([]int64, error) {
	var adminIDs []int64
	
	// Разделяем по запятым
	parts := strings.Split(adminIDsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid admin ID '%s': %w", part, err)
		}
		
		adminIDs = append(adminIDs, id)
	}
	
	return adminIDs, nil
}

// getDefaultSystemPrompt возвращает системный промпт по умолчанию
func getDefaultSystemPrompt() string {
	return `Ты - помощник по отношениям Lovifyy Bot. 
Твоя задача - помогать парам улучшать их отношения через:
- Ответы на вопросы о отношениях
- Предложение упражнений для пар
- Поддержку ведения дневника отношений
- Мотивационные сообщения

Отвечай дружелюбно, с пониманием и профессионально.
Используй эмодзи для лучшего восприятия.`
}

// ValidateTelegramConfig проверяет корректность конфигурации Telegram
func (tc TelegramConfig) Validate() error {
	if tc.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}
	
	if tc.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	if tc.RetryCount <= 0 {
		return fmt.Errorf("retry count must be positive")
	}
	
	// Проверяем формат токена (базовая проверка)
	if !strings.Contains(tc.BotToken, ":") {
		return fmt.Errorf("invalid bot token format")
	}
	
	return nil
}
