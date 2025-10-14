package tests

import (
	"testing"
	"time"
	"github.com/godofphonk/lovifyy-bot/internal/models"
)

func TestUserManager(t *testing.T) {
	// Создаем новый user manager
	adminIDs := []int64{123456789}
	userManager := models.NewUserManager(adminIDs)
	
	userID := int64(12345)
	
	// Тестируем состояния
	userManager.SetState(userID, "chat")
	state := userManager.GetState(userID)
	if state != "chat" {
		t.Errorf("Ожидали состояние 'chat', получили '%s'", state)
	}
	
	// Тестируем админов
	if !userManager.IsAdmin(123456789) {
		t.Error("Пользователь должен быть админом")
	}
	
	if userManager.IsAdmin(userID) {
		t.Error("Пользователь не должен быть админом")
	}
}

func TestRateLimiting(t *testing.T) {
	adminIDs := []int64{}
	userManager := models.NewUserManager(adminIDs)
	
	userID := int64(12345)
	limit := 100 * time.Millisecond
	
	// Первый запрос должен пройти
	if userManager.IsRateLimited(userID, limit) {
		t.Error("Первый запрос должен быть разрешен")
	}
	
	// Второй запрос сразу же должен быть заблокирован
	if !userManager.IsRateLimited(userID, limit) {
		t.Error("Второй запрос должен быть заблокирован")
	}
	
	// После паузы должен пройти
	time.Sleep(limit + 10*time.Millisecond)
	if userManager.IsRateLimited(userID, limit) {
		t.Error("Запрос после паузы должен быть разрешен")
	}
}

func TestMessageValidation(t *testing.T) {
	// Тест будет расширен когда функции валидации станут публичными
	// Пока что проверяем, что rate limiter работает
	t.Log("Тесты валидации сообщений будут добавлены в будущем")
}
