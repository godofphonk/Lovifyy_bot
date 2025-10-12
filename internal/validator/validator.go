package validator

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
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

// ValidateMessage валидирует сообщение пользователя
func (v *Validator) ValidateMessage(message string) ValidationResult {
	var errors []ValidationError
	
	// Проверка на пустоту
	if strings.TrimSpace(message) == "" {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Message cannot be empty",
			Code:    "EMPTY_MESSAGE",
		})
	}
	
	// Проверка длины
	if utf8.RuneCountInString(message) > v.maxMessageLength {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: fmt.Sprintf("Message too long (max %d characters)", v.maxMessageLength),
			Code:    "MESSAGE_TOO_LONG",
		})
	}
	
	// Проверка на вредоносный контент
	if v.containsMaliciousContent(message) {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Message contains potentially malicious content",
			Code:    "MALICIOUS_CONTENT",
		})
	}
	
	// Проверка на спам
	if v.isSpam(message) {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Message appears to be spam",
			Code:    "SPAM_DETECTED",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateCommand валидирует команду
func (v *Validator) ValidateCommand(command string) ValidationResult {
	var errors []ValidationError
	
	// Проверка на пустоту
	if strings.TrimSpace(command) == "" {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "Command cannot be empty",
			Code:    "EMPTY_COMMAND",
		})
		return ValidationResult{Valid: false, Errors: errors}
	}
	
	// Удаляем префикс /
	cleanCommand := strings.TrimPrefix(command, "/")
	
	// Проверка на разрешенные команды
	if !v.isAllowedCommand(cleanCommand) {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: fmt.Sprintf("Command '%s' is not allowed", cleanCommand),
			Code:    "COMMAND_NOT_ALLOWED",
		})
	}
	
	// Проверка формата команды
	if !v.isValidCommandFormat(cleanCommand) {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "Invalid command format",
			Code:    "INVALID_COMMAND_FORMAT",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateUserID валидирует ID пользователя
func (v *Validator) ValidateUserID(userID int64) ValidationResult {
	var errors []ValidationError
	
	if userID <= 0 {
		errors = append(errors, ValidationError{
			Field:   "user_id",
			Message: "User ID must be positive",
			Code:    "INVALID_USER_ID",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// SanitizeMessage очищает сообщение от потенциально опасного контента
func (v *Validator) SanitizeMessage(message string) string {
	// Удаляем HTML теги если включена санитизация
	if v.sanitizeHTML {
		message = html.EscapeString(message)
	}
	
	// Удаляем лишние пробелы
	message = strings.TrimSpace(message)
	message = regexp.MustCompile(`\s+`).ReplaceAllString(message, " ")
	
	// Удаляем потенциально опасные символы
	message = v.removeMaliciousChars(message)
	
	return message
}

// SanitizeCommand очищает команду
func (v *Validator) SanitizeCommand(command string) string {
	// Удаляем лишние пробелы
	command = strings.TrimSpace(command)
	
	// Приводим к нижнему регистру
	command = strings.ToLower(command)
	
	// Удаляем потенциально опасные символы
	command = regexp.MustCompile(`[^a-zA-Z0-9_/\s]`).ReplaceAllString(command, "")
	
	return command
}

// containsMaliciousContent проверяет на вредоносный контент
func (v *Validator) containsMaliciousContent(message string) bool {
	maliciousPatterns := []string{
		`<script`,
		`javascript:`,
		`data:text/html`,
		`vbscript:`,
		`onload=`,
		`onerror=`,
		`onclick=`,
		`eval\(`,
		`document\.`,
		`window\.`,
	}
	
	lowerMessage := strings.ToLower(message)
	for _, pattern := range maliciousPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerMessage); matched {
			return true
		}
	}
	
	return false
}

// isSpam проверяет на спам
func (v *Validator) isSpam(message string) bool {
	// Проверка на повторяющиеся символы
	if v.hasExcessiveRepetition(message) {
		return true
	}
	
	// Проверка на слишком много заглавных букв
	if v.hasExcessiveCaps(message) {
		return true
	}
	
	// Проверка на подозрительные URL
	if v.hasSuspiciousURLs(message) {
		return true
	}
	
	return false
}

// hasExcessiveRepetition проверяет на чрезмерное повторение
func (v *Validator) hasExcessiveRepetition(message string) bool {
	// Проверяем повторение одного символа более 10 раз подряд
	// Простая проверка без регекса
	if len(message) < 10 {
		return false
	}
	
	for i := 0; i < len(message)-10; i++ {
		char := message[i]
		count := 1
		for j := i + 1; j < len(message) && message[j] == char; j++ {
			count++
			if count > 10 {
				return true
			}
		}
	}
	return false
}

// hasExcessiveCaps проверяет на чрезмерное использование заглавных букв
func (v *Validator) hasExcessiveCaps(message string) bool {
	if len(message) < 10 {
		return false
	}
	
	upperCount := 0
	for _, r := range message {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		}
	}
	
	// Если более 70% символов в верхнем регистре
	return float64(upperCount)/float64(len(message)) > 0.7
}

// hasSuspiciousURLs проверяет на подозрительные URL
func (v *Validator) hasSuspiciousURLs(message string) bool {
	// Простая проверка на подозрительные домены
	suspiciousDomains := []string{
		"bit.ly",
		"tinyurl.com",
		"t.co",
		"goo.gl",
	}
	
	lowerMessage := strings.ToLower(message)
	for _, domain := range suspiciousDomains {
		if strings.Contains(lowerMessage, domain) {
			return true
		}
	}
	
	return false
}

// isAllowedCommand проверяет, разрешена ли команда
func (v *Validator) isAllowedCommand(command string) bool {
	for _, allowed := range v.allowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// isValidCommandFormat проверяет формат команды
func (v *Validator) isValidCommandFormat(command string) bool {
	// Команда должна содержать только буквы, цифры и подчеркивания
	pattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return pattern.MatchString(command)
}

// removeMaliciousChars удаляет потенциально опасные символы
func (v *Validator) removeMaliciousChars(message string) string {
	// Удаляем управляющие символы кроме переноса строки и табуляции
	result := make([]rune, 0, len(message))
	for _, r := range message {
		if r == '\n' || r == '\t' || r >= 32 {
			result = append(result, r)
		}
	}
	return string(result)
}

// GetDefaultConfig возвращает конфигурацию по умолчанию
func GetDefaultConfig() Config {
	return Config{
		MaxMessageLength: 4000,
		AllowedCommands: []string{
			"start", "help", "menu", "admin", "notify", "setweek",
			"chat", "diary", "exercises", "settings",
		},
		SanitizeHTML: true,
	}
}
