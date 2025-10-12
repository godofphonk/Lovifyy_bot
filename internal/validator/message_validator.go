package validator

import (
	"strings"
	"unicode/utf8"
)

// ValidateMessage валидирует сообщение пользователя
func (v *Validator) ValidateMessage(message string) ValidationResult {
	var errors []ValidationError
	
	// Проверка на пустоту
	if strings.TrimSpace(message) == "" {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Сообщение не может быть пустым",
			Code:    "EMPTY_MESSAGE",
		})
	}
	
	// Проверка длины
	if utf8.RuneCountInString(message) > v.maxMessageLength {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Сообщение слишком длинное",
			Code:    "MESSAGE_TOO_LONG",
		})
	}
	
	// Проверка на вредоносный контент
	if v.containsMaliciousContent(message) {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Сообщение содержит недопустимый контент",
			Code:    "MALICIOUS_CONTENT",
		})
	}
	
	// Проверка на спам
	if v.isSpam(message) {
		errors = append(errors, ValidationError{
			Field:   "message",
			Message: "Сообщение похоже на спам",
			Code:    "SPAM_DETECTED",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
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
		`<iframe`,
		`<object`,
		`<embed`,
	}
	
	messageLower := strings.ToLower(message)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(messageLower, pattern) {
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
	
	// Проверка на чрезмерное использование заглавных букв
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
	if len(message) < 10 {
		return false
	}
	
	count := 1
	for i := 1; i < len(message); i++ {
		if message[i] == message[i-1] {
			count++
			if count > 10 {
				return true
			}
		} else {
			count = 1
		}
	}
	
	return false
}

// hasExcessiveCaps проверяет на чрезмерное использование заглавных букв
func (v *Validator) hasExcessiveCaps(message string) bool {
	if len(message) < 10 {
		return false
	}
	
	capsCount := 0
	for _, r := range message {
		if r >= 'A' && r <= 'Z' {
			capsCount++
		}
	}
	
	// Если более 70% символов в верхнем регистре
	return float64(capsCount)/float64(len(message)) > 0.7
}

// hasSuspiciousURLs проверяет на подозрительные URL
func (v *Validator) hasSuspiciousURLs(message string) bool {
	// Простая проверка на подозрительные домены
	suspiciousDomains := []string{
		"bit.ly",
		"tinyurl.com",
		"t.co",
		"goo.gl",
		"ow.ly",
		"short.link",
	}
	
	messageLower := strings.ToLower(message)
	for _, domain := range suspiciousDomains {
		if strings.Contains(messageLower, domain) {
			return true
		}
	}
	
	return false
}
