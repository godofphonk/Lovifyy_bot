package tests

import (
	"testing"
	"time"

	"Lovifyy_bot/internal/models"
	"Lovifyy_bot/tests/mocks"
)

func TestNotificationServiceIntegration(t *testing.T) {
	// Создаем моки
	mockBot := &mocks.MockTelegramBot{}
	mockAI := &mocks.MockAIClient{
		GenerateFunc: func(prompt string) (string, error) {
			return "Generated notification message", nil
		},
	}

	// Создаем сервис уведомлений (пропускаем пока что)
	_ = mockBot
	_ = mockAI
	
	// TODO: Исправить после обновления интерфейсов
	t.Skip("Skipping until interfaces are updated")
}

func TestUserManagerIntegration(t *testing.T) {
	adminIDs := []int64{123456789}
	userManager := models.NewUserManager(adminIDs)

	userID := int64(12345)

	// Тестируем полный цикл состояний
	userManager.SetState(userID, "chat")
	if state := userManager.GetState(userID); state != "chat" {
		t.Errorf("Expected state 'chat', got '%s'", state)
	}

	// Тестируем состояние с данными
	userManager.SetStateData(userID, "diary", "week_1")
	state, data := userManager.GetStateData(userID)
	if state != "diary" || data != "week_1" {
		t.Errorf("Expected state 'diary' with data 'week_1', got '%s' with '%s'", state, data)
	}

	// Тестируем очистку состояния
	userManager.ClearState(userID)
	if state := userManager.GetState(userID); state != "" {
		t.Errorf("Expected empty state after clear, got '%s'", state)
	}
}

func TestRateLimitingIntegration(t *testing.T) {
	userManager := models.NewUserManager([]int64{})
	userID := int64(12345)
	limit := 100 * time.Millisecond

	// Первый запрос должен пройти
	if userManager.IsRateLimited(userID, limit) {
		t.Error("First request should not be rate limited")
	}

	// Второй запрос сразу должен быть заблокирован
	if !userManager.IsRateLimited(userID, limit) {
		t.Error("Second immediate request should be rate limited")
	}

	// После паузы должен пройти
	time.Sleep(limit + 10*time.Millisecond)
	if userManager.IsRateLimited(userID, limit) {
		t.Error("Request after pause should not be rate limited")
	}
}

func TestHistoryManagerIntegration(t *testing.T) {
	mockHistory := &mocks.MockHistoryManager{}
	userID := int64(12345)

	// Тестируем сохранение сообщения
	err := mockHistory.SaveMessage(userID, "testuser", "Hello", "Hi there!", "gpt-4o-mini")
	if err != nil {
		t.Errorf("Failed to save message: %v", err)
	}

	// Проверяем, что сообщение сохранилось
	if len(mockHistory.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(mockHistory.Messages))
	}

	// Тестируем получение истории
	history, err := mockHistory.GetUserHistory(userID, 10)
	if err != nil {
		t.Errorf("Failed to get user history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 message in history, got %d", len(history))
	}

	if history[0].Message != "Hello" {
		t.Errorf("Expected message 'Hello', got '%s'", history[0].Message)
	}
}

func TestDiaryIntegration(t *testing.T) {
	mockHistory := &mocks.MockHistoryManager{}
	userID := int64(12345)

	// Тестируем сохранение записи в дневник
	err := mockHistory.SaveDiaryEntry(userID, "testuser", "Today was great!", 1, "personal")
	if err != nil {
		t.Errorf("Failed to save diary entry: %v", err)
	}

	// Проверяем, что запись сохранилась
	if len(mockHistory.Entries) != 1 {
		t.Errorf("Expected 1 diary entry, got %d", len(mockHistory.Entries))
	}

	// Тестируем получение записей
	entries, err := mockHistory.GetDiaryEntries(userID, 1, "personal")
	if err != nil {
		t.Errorf("Failed to get diary entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 diary entry, got %d", len(entries))
	}

	if entries[0].Entry != "Today was great!" {
		t.Errorf("Expected entry 'Today was great!', got '%s'", entries[0].Entry)
	}
}

func TestFullWorkflowIntegration(t *testing.T) {
	// Создаем все компоненты
	adminIDs := []int64{123456789}
	userManager := models.NewUserManager(adminIDs)
	mockHistory := &mocks.MockHistoryManager{}
	mockAI := &mocks.MockAIClient{
		GenerateFunc: func(prompt string) (string, error) {
			return "AI response to: " + prompt, nil
		},
	}

	userID := int64(12345)

	// Симулируем полный workflow пользователя
	t.Run("User starts chat", func(t *testing.T) {
		userManager.SetState(userID, "chat")
		state := userManager.GetState(userID)
		if state != "chat" {
			t.Errorf("Expected chat state, got %s", state)
		}
	})

	t.Run("User sends message", func(t *testing.T) {
		message := "How can I improve my relationship?"
		
		// Сохраняем сообщение
		err := mockHistory.SaveMessage(userID, "testuser", message, "", "gpt-4o-mini")
		if err != nil {
			t.Errorf("Failed to save user message: %v", err)
		}

		// Генерируем ответ AI
		response, err := mockAI.Generate("System prompt + " + message)
		if err != nil {
			t.Errorf("Failed to generate AI response: %v", err)
		}

		// Сохраняем ответ
		err = mockHistory.SaveMessage(userID, "testuser", message, response, "gpt-4o-mini")
		if err != nil {
			t.Errorf("Failed to save AI response: %v", err)
		}

		// Проверяем историю
		history, err := mockHistory.GetUserHistory(userID, 10)
		if err != nil {
			t.Errorf("Failed to get history: %v", err)
		}

		if len(history) < 1 {
			t.Error("Expected at least 1 message in history")
		}
	})

	t.Run("User switches to diary", func(t *testing.T) {
		userManager.SetState(userID, "diary")
		userManager.SetStateData(userID, "diary", "week_1")

		state, data := userManager.GetStateData(userID)
		if state != "diary" || data != "week_1" {
			t.Errorf("Expected diary state with week_1 data, got %s with %s", state, data)
		}
	})

	t.Run("User writes diary entry", func(t *testing.T) {
		entry := "Today I learned about communication in relationships"
		err := mockHistory.SaveDiaryEntry(userID, "testuser", entry, 1, "personal")
		if err != nil {
			t.Errorf("Failed to save diary entry: %v", err)
		}

		entries, err := mockHistory.GetDiaryEntries(userID, 1, "personal")
		if err != nil {
			t.Errorf("Failed to get diary entries: %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("Expected 1 diary entry, got %d", len(entries))
		}
	})

	t.Run("Admin sends notification", func(t *testing.T) {
		if !userManager.IsAdmin(123456789) {
			t.Error("User should be admin")
		}

		// TODO: Исправить после обновления интерфейсов
		t.Skip("Skipping notification test until interfaces are updated")
	})

	t.Logf("Full workflow test completed successfully")
}
