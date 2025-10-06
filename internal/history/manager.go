package history

import (
	"encoding/json"
	"fmt"
	"log"
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

// DiaryEntry представляет одну запись в дневнике
type DiaryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Entry     string    `json:"entry"`
	Mood      string    `json:"mood,omitempty"`     // настроение (опционально)
	Tags      []string  `json:"tags,omitempty"`     // теги (опционально)
}

// Manager управляет историей переписки и дневниками
type Manager struct {
	historyDir string
	diaryDir   string
}

// NewManager создает новый менеджер истории
func NewManager() *Manager {
	historyDir := "chat_history"
	diaryDir := "diary_entries"
	os.MkdirAll(historyDir, 0755)
	os.MkdirAll(diaryDir, 0755)
	return &Manager{
		historyDir: historyDir,
		diaryDir:   diaryDir,
	}
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

	// Ограничиваем историю последними 100 сообщениями для экономии места
	const maxHistorySize = 100
	if len(history) > maxHistorySize {
		history = history[len(history)-maxHistorySize:]
		log.Printf("История пользователя %d обрезана до %d сообщений", userID, maxHistorySize)
	}

	// Сохраняем обновленную историю
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		log.Printf("Ошибка сериализации истории для пользователя %d: %v", userID, err)
		return fmt.Errorf("ошибка сериализации истории: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Ошибка записи истории для пользователя %d: %v", userID, err)
		return fmt.Errorf("ошибка записи истории: %w", err)
	}

	return nil
}

// GetUserHistory получает историю пользователя
func (m *Manager) GetUserHistory(userID int64, limit int) ([]ChatMessage, error) {
	filename := filepath.Join(m.historyDir, fmt.Sprintf("user_%d.json", userID))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChatMessage{}, nil // Пустая история, если файл не найден
		}
		log.Printf("Ошибка чтения файла истории для пользователя %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка чтения истории: %w", err)
	}

	var history []ChatMessage
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("Ошибка парсинга истории для пользователя %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка парсинга истории: %w", err)
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

// SaveDiaryEntry сохраняет запись в дневник
func (m *Manager) SaveDiaryEntry(userID int64, username, entry string) error {
	diaryEntry := DiaryEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Entry:     entry,
	}

	// Создаем файл дневника для каждого пользователя
	filename := filepath.Join(m.diaryDir, fmt.Sprintf("diary_%d.json", userID))
	
	// Читаем существующие записи дневника
	var diary []DiaryEntry
	if data, err := os.ReadFile(filename); err == nil {
		json.Unmarshal(data, &diary)
	}

	// Добавляем новую запись
	diary = append(diary, diaryEntry)

	// Сохраняем обновленный дневник
	data, err := json.MarshalIndent(diary, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// GetUserDiary получает записи дневника пользователя
func (m *Manager) GetUserDiary(userID int64, limit int) ([]DiaryEntry, error) {
	filename := filepath.Join(m.diaryDir, fmt.Sprintf("diary_%d.json", userID))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []DiaryEntry{}, nil // Возвращаем пустой дневник, если файла нет
		}
		return nil, err
	}

	var diary []DiaryEntry
	err = json.Unmarshal(data, &diary)
	if err != nil {
		return nil, err
	}

	// Если лимит указан, возвращаем последние записи
	if limit > 0 && len(diary) > limit {
		return diary[len(diary)-limit:], nil
	}

	return diary, nil
}

// GetDiaryStats получает статистику дневника
func (m *Manager) GetDiaryStats(userID int64) (int, time.Time, error) {
	diary, err := m.GetUserDiary(userID, 0)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(diary) == 0 {
		return 0, time.Time{}, nil
	}

	totalEntries := len(diary)
	firstEntry := diary[0].Timestamp

	return totalEntries, firstEntry, nil
}

// ClearUserDiary очищает дневник конкретного пользователя
func (m *Manager) ClearUserDiary(userID int64) error {
	filename := filepath.Join(m.diaryDir, fmt.Sprintf("diary_%d.json", userID))
	return os.Remove(filename)
}
