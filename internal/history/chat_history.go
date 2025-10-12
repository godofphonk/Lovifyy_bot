package history

import (
	"fmt"
	"strings"
	"time"
)

// OpenAIMessage представляет сообщение в формате OpenAI API
type OpenAIMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // текст сообщения
}

// SaveMessage сохраняет сообщение в историю
func (m *Manager) SaveMessage(userID int64, username, message, response, model string) error {
	chatMsg := ChatMessage{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Message:   message,
		Response:  m.cleanResponse(response),
		Model:     model,
	}

	filename := m.getUserChatFile(userID)
	
	// Загружаем существующую историю
	var history []ChatMessage
	if err := m.loadFromFile(filename, &history); err != nil {
		return fmt.Errorf("failed to load existing history: %w", err)
	}

	// Добавляем новое сообщение
	history = append(history, chatMsg)

	// Ограничиваем размер истории (например, последние 1000 сообщений)
	const maxHistorySize = 1000
	if len(history) > maxHistorySize {
		history = history[len(history)-maxHistorySize:]
	}

	// Сохраняем обновленную историю
	return m.saveToFile(filename, history)
}

// GetUserHistory получает историю пользователя
func (m *Manager) GetUserHistory(userID int64, limit int) ([]ChatMessage, error) {
	filename := m.getUserChatFile(userID)
	
	var history []ChatMessage
	if err := m.loadFromFile(filename, &history); err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	// Применяем лимит если указан
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history, nil
}

// GetRecentContext получает недавний контекст для ИИ
func (m *Manager) GetRecentContext(userID int64, contextLimit int) string {
	history, err := m.GetUserHistory(userID, contextLimit)
	if err != nil || len(history) == 0 {
		return ""
	}

	var context strings.Builder
	for _, msg := range history {
		context.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n\n", msg.Message, msg.Response))
	}

	return context.String()
}

// ClearUserHistory очищает историю конкретного пользователя
func (m *Manager) ClearUserHistory(userID int64) error {
	filename := m.getUserChatFile(userID)
	return m.removeFile(filename)
}

// GetOpenAIHistory возвращает историю в формате OpenAI с ограничением
func (m *Manager) GetOpenAIHistory(userID int64, systemPrompt string, limit int) ([]OpenAIMessage, error) {
	// Загружаем обычную историю
	messages, err := m.GetUserHistory(userID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get user history: %w", err)
	}

	var openAIMessages []OpenAIMessage

	// Добавляем системный промпт если указан
	if systemPrompt != "" {
		openAIMessages = append(openAIMessages, OpenAIMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Применяем лимит к сообщениям пользователя
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	// Конвертируем в формат OpenAI
	for _, msg := range messages {
		// Добавляем сообщение пользователя
		openAIMessages = append(openAIMessages, OpenAIMessage{
			Role:    "user",
			Content: msg.Message,
		})

		// Добавляем ответ ассистента
		openAIMessages = append(openAIMessages, OpenAIMessage{
			Role:    "assistant",
			Content: msg.Response,
		})
	}

	return openAIMessages, nil
}

// SaveOpenAIMessage сохраняет сообщение в формате совместимом с OpenAI
func (m *Manager) SaveOpenAIMessage(userID int64, username, userMessage, assistantResponse, model string) error {
	return m.SaveMessage(userID, username, userMessage, assistantResponse, model)
}
