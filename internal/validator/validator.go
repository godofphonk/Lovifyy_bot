package validator

import (
	"fmt"
)

// Validator структура для валидации данных
type Validator struct {
	maxMessageLength int
	allowedCommands  []string
	sanitizeHTML     bool
}

// Config конфигурация валидатора
type Config struct {
	MaxMessageLength int      `json:"max_message_length"`
	AllowedCommands  []string `json:"allowed_commands"`
	SanitizeHTML     bool     `json:"sanitize_html"`
}

// ValidationError ошибка валидации
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidationResult результат валидации
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewValidator создает новый валидатор
func NewValidator(config Config) *Validator {
	return &Validator{
		maxMessageLength: config.MaxMessageLength,
		allowedCommands:  config.AllowedCommands,
		sanitizeHTML:     config.SanitizeHTML,
	}
}
