package validator

import (
	"html"
	"strings"
	"unicode"
)

// SanitizeMessage очищает сообщение от потенциально опасного контента
func (v *Validator) SanitizeMessage(message string) string {
	// Удаляем HTML теги если включена санитизация
	if v.sanitizeHTML {
		message = html.EscapeString(message)
	}
	
	// Удаляем потенциально опасные символы
	message = v.removeMaliciousChars(message)
	
	// Нормализуем пробелы
	message = v.normalizeWhitespace(message)
	
	return message
}

// SanitizeCommand очищает команду
func (v *Validator) SanitizeCommand(command string) string {
	// Удаляем лишние пробелы
	command = strings.TrimSpace(command)
	
	// Приводим к нижнему регистру
	command = strings.ToLower(command)
	
	// Удаляем потенциально опасные символы
	command = v.removeMaliciousChars(command)
	
	return command
}

// SanitizeUsername очищает имя пользователя
func (v *Validator) SanitizeUsername(username string) string {
	// Удаляем лишние пробелы
	username = strings.TrimSpace(username)
	
	// Удаляем потенциально опасные символы
	username = v.removeMaliciousChars(username)
	
	// Ограничиваем длину
	if len(username) > 32 {
		username = username[:32]
	}
	
	return username
}

// removeMaliciousChars удаляет потенциально опасные символы
func (v *Validator) removeMaliciousChars(message string) string {
	// Удаляем управляющие символы кроме переноса строки и табуляции
	result := make([]rune, 0, len(message))
	for _, r := range message {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			continue
		}
		result = append(result, r)
	}
	
	return string(result)
}

// normalizeWhitespace нормализует пробельные символы
func (v *Validator) normalizeWhitespace(message string) string {
	// Заменяем множественные пробелы одним
	message = strings.Join(strings.Fields(message), " ")
	
	// Удаляем лишние переносы строк
	lines := strings.Split(message, "\n")
	var normalizedLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" || len(normalizedLines) == 0 || normalizedLines[len(normalizedLines)-1] != "" {
			normalizedLines = append(normalizedLines, line)
		}
	}
	
	return strings.Join(normalizedLines, "\n")
}

// RemoveHTMLTags удаляет HTML теги из текста
func (v *Validator) RemoveHTMLTags(text string) string {
	// Простое удаление HTML тегов
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

// EscapeSpecialChars экранирует специальные символы
func (v *Validator) EscapeSpecialChars(text string) string {
	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
	}
	
	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}
	
	return text
}
