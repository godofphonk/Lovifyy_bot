package validator

// ValidateUserID валидирует ID пользователя
func (v *Validator) ValidateUserID(userID int64) ValidationResult {
	var errors []ValidationError
	
	if userID <= 0 {
		errors = append(errors, ValidationError{
			Field:   "user_id",
			Message: "ID пользователя должен быть положительным числом",
			Code:    "INVALID_USER_ID",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateUsername валидирует имя пользователя
func (v *Validator) ValidateUsername(username string) ValidationResult {
	var errors []ValidationError
	
	// Имя пользователя может быть пустым (не все пользователи Telegram имеют username)
	if username == "" {
		return ValidationResult{Valid: true}
	}
	
	// Проверка длины
	if len(username) > 32 {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя слишком длинное",
			Code:    "USERNAME_TOO_LONG",
		})
	}
	
	// Проверка на недопустимые символы
	if v.containsMaliciousContent(username) {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя содержит недопустимые символы",
			Code:    "INVALID_USERNAME_CHARS",
		})
	}
	
	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
