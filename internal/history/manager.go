package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ChatMessage представляет одно сообщение в истории
type ChatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	Model     string    `json:"model"`
}

// Manager управляет историей переписки
type Manager struct {
	historyDir string
}

// NewManager создает новый менеджер истории
func NewManager() *Manager {
	historyDir := "chat_history"
	os.MkdirAll(historyDir, 0755)
	return &Manager{historyDir: historyDir}
}

// SaveMessage сохраняет сообщение в историю
func (m *Manager) SaveMessage(userID int64, username, message, response, model string) error {
	chatMsg := ChatMessage{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Message:   message,
		Response:  response,
		Model:     model,
	}

	// Создаем файл для каждого пользователя
	filename := filepath.Join(m.historyDir, fmt.Sprintf("user_%d.json", userID))
	
	// Читаем существующую историю
	var history []ChatMessage
	if data, err := os.ReadFile(filename); err == nil {
		json.Unmarshal(data, &history)
	}

	// Добавляем новое сообщение
	history = append(history, chatMsg)

	// Сохраняем обновленную историю
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// GetUserHistory получает историю пользователя
func (m *Manager) GetUserHistory(userID int64, limit int) ([]ChatMessage, error) {
	filename := filepath.Join(m.historyDir, fmt.Sprintf("user_%d.json", userID))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return []ChatMessage{}, nil // Пустая история, если файл не найден
	}

	var history []ChatMessage
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	// Возвращаем последние N сообщений
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:], nil
	}

	return history, nil
}

// GetRecentContext получает недавний контекст для ИИ
func (m *Manager) GetRecentContext(userID int64, contextLimit int) string {
	history, err := m.GetUserHistory(userID, contextLimit)
	if err != nil || len(history) == 0 {
		return ""
	}

	context := "Предыдущие сообщения:\n"
	for _, msg := range history {
		context += fmt.Sprintf("Пользователь: %s\nБот: %s\n\n", msg.Message, msg.Response)
	}

	return context
}

// GetStats получает статистику использования
func (m *Manager) GetStats(userID int64) (int, time.Time, error) {
	history, err := m.GetUserHistory(userID, 0)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(history) == 0 {
		return 0, time.Time{}, nil
	}

	totalMessages := len(history)
	firstMessage := history[0].Timestamp

	return totalMessages, firstMessage, nil
}

// ClearUserHistory очищает историю конкретного пользователя
func (m *Manager) ClearUserHistory(userID int64) error {
	filename := filepath.Join(m.historyDir, fmt.Sprintf("user_%d.json", userID))
	return os.Remove(filename)
}
