package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// OpenAIConfig конфигурация OpenAI API
type OpenAIConfig struct {
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
}

// LoadOpenAIConfig загружает конфигурацию OpenAI из переменных окружения
func LoadOpenAIConfig() (OpenAIConfig, error) {
	config := OpenAIConfig{
		Model:       "gpt-3.5-turbo", // значение по умолчанию
		BaseURL:     "https://api.openai.com/v1", // значение по умолчанию
		MaxTokens:   1500,            // значение по умолчанию
		Temperature: 0.7,             // значение по умолчанию
		Timeout:     30 * time.Second, // значение по умолчанию
	}

	// Загружаем API ключ
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return config, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}
	config.APIKey = apiKey

	// Загружаем модель
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		config.Model = model
	}

	// Загружаем базовый URL
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	// Загружаем максимальное количество токенов
	if maxTokensStr := os.Getenv("OPENAI_MAX_TOKENS"); maxTokensStr != "" {
		if maxTokens, err := strconv.Atoi(maxTokensStr); err == nil && maxTokens > 0 {
			config.MaxTokens = maxTokens
		}
	}

	// Загружаем температуру
	if tempStr := os.Getenv("OPENAI_TEMPERATURE"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil && temp >= 0 && temp <= 2 {
			config.Temperature = temp
		}
	}

	// Загружаем timeout
	if timeoutStr := os.Getenv("OPENAI_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	return config, nil
}

// ValidateOpenAIConfig проверяет корректность конфигурации OpenAI
func (oc OpenAIConfig) Validate() error {
	if oc.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	
	if oc.Model == "" {
		return fmt.Errorf("model is required")
	}
	
	if oc.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	
	if oc.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	
	if oc.Temperature < 0 || oc.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	
	if oc.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	// Проверяем формат API ключа (базовая проверка)
	if len(oc.APIKey) < 10 {
		return fmt.Errorf("API key seems too short")
	}
	
	return nil
}

// GetSupportedModels возвращает список поддерживаемых моделей
func GetSupportedModels() []string {
	return []string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-4-turbo-preview",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
	}
}

// IsModelSupported проверяет, поддерживается ли модель
func IsModelSupported(model string) bool {
	supportedModels := GetSupportedModels()
	for _, supported := range supportedModels {
		if model == supported {
			return true
		}
	}
	return false
}
