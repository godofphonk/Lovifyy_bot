package tests

import (
	"testing"
	"Lovifyy_bot/internal/bot"
)

func TestRateLimiter(t *testing.T) {
	// Создаем новый rate limiter
	rateLimiter := bot.NewRateLimiter()
	
	userID := int64(12345)
	
	// Первый запрос должен пройти
	if !rateLimiter.IsAllowed(userID) {
		t.Error("Первый запрос должен быть разрешен")
	}
	
	// Второй запрос сразу же должен быть заблокирован
	if rateLimiter.IsAllowed(userID) {
		t.Error("Второй запрос должен быть заблокирован")
	}
}

func TestMessageValidation(t *testing.T) {
	// Тест будет расширен когда функции валидации станут публичными
	// Пока что проверяем, что rate limiter работает
	t.Log("Тесты валидации сообщений будут добавлены в будущем")
}
