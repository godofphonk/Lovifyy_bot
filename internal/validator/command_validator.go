package validator

import (
	"regexp"
	"strings"
)

// ValidateCommand валидирует команду
func (v *Validator) ValidateCommand(command string) ValidationResult {
	var errors []ValidationError
	
	// Проверка на пустоту
	if strings.TrimSpace(command) == "" {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "Команда не может быть пустой",
			Code:    "EMPTY_COMMAND",
		})
		return ValidationResult{
			Valid:  false,
			Errors: errors,
		}
	}
	
	// Проверка формата команды
	if !v.isValidCommandFormat(command) {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "Неверный формат команды",
			Code:    "INVALID_COMMAND_FORMAT",
		})
	}
	
	// Проверка на разрешенные команды
	if len(v.allowedCommands) > 0 && !v.isAllowedCommand(command) {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "Команда не разрешена",
			Code:    "COMMAND_NOT_ALLOWED",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
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
