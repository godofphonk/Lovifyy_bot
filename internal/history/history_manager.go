package history

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	Week      int       `json:"week"`               // номер недели (1-4)
	Type      string    `json:"type"`               // тип записи: questions, joint, personal
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
	historyDir := "data/chats"
	diaryDir := "data/diaries"
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

	// Создаем папку для пользователя
	userDir := filepath.Join(m.historyDir, fmt.Sprintf("user_%d", userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки пользователя: %w", err)
	}
	
	filename := filepath.Join(userDir, "chat.json")
	
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
	userDir := filepath.Join(m.historyDir, fmt.Sprintf("user_%d", userID))
	filename := filepath.Join(userDir, "chat.json")
	
	// Для совместимости со старым форматом
	oldFilename := filepath.Join(m.historyDir, fmt.Sprintf("user_%d.json", userID))
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if _, err := os.Stat(oldFilename); err == nil {
			filename = oldFilename
		}
	}
	
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
	userDir := filepath.Join(m.historyDir, fmt.Sprintf("user_%d", userID))
	filename := filepath.Join(userDir, "chat.json")
	return os.Remove(filename)
}

// GetDiaryEntriesByWeek получает записи дневника для конкретной недели (только questions и personal)
func (m *Manager) GetDiaryEntriesByWeek(userID int64, week int) ([]DiaryEntry, error) {
	var allWeekEntries []DiaryEntry
	
	// Читаем записи из папки "diary_questions"
	questionsFile := filepath.Join(m.diaryDir, "diary_questions", fmt.Sprintf("user_%d.json", userID))
	if data, err := os.ReadFile(questionsFile); err == nil {
		var questionsEntries []DiaryEntry
		if err := json.Unmarshal(data, &questionsEntries); err == nil {
			for _, entry := range questionsEntries {
				if entry.Week == week {
					allWeekEntries = append(allWeekEntries, entry)
				}
			}
		}
	}
	
	// Читаем записи из папки "diary_thoughts"
	thoughtsFile := filepath.Join(m.diaryDir, "diary_thoughts", fmt.Sprintf("user_%d.json", userID))
	if data, err := os.ReadFile(thoughtsFile); err == nil {
		var thoughtsEntries []DiaryEntry
		if err := json.Unmarshal(data, &thoughtsEntries); err == nil {
			for _, entry := range thoughtsEntries {
				if entry.Week == week {
					allWeekEntries = append(allWeekEntries, entry)
				}
			}
		}
	}
	
	// Для совместимости со старыми записями - читаем из старых файлов
	oldFiles := []string{
		filepath.Join(m.diaryDir, fmt.Sprintf("diary_questions_%d.json", userID)),
		filepath.Join(m.diaryDir, fmt.Sprintf("diary_personal_%d.json", userID)),
		filepath.Join(m.diaryDir, fmt.Sprintf("diary_%d.json", userID)),
	}
	
	for _, oldFile := range oldFiles {
		if data, err := os.ReadFile(oldFile); err == nil {
			var oldEntries []DiaryEntry
			if err := json.Unmarshal(data, &oldEntries); err == nil {
				for _, entry := range oldEntries {
					if entry.Week == week && (entry.Type == "questions" || entry.Type == "personal") {
						allWeekEntries = append(allWeekEntries, entry)
					}
				}
			}
		}
	}
	
	return allWeekEntries, nil
}

// SaveDiaryEntry сохраняет запись в дневник (в отдельные файлы по типам)
func (m *Manager) SaveDiaryEntry(userID int64, username, entry string, week int, entryType string) error {
	diaryEntry := DiaryEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Entry:     entry,
		Week:      week,
		Type:      entryType,
	}

	// Определяем папку и файл в зависимости от типа записи (без гендера для совместимости)
	var typeDir, filename string
	switch entryType {
	case "questions":
		typeDir = filepath.Join(m.diaryDir, "diary_questions")
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	case "joint":
		typeDir = filepath.Join(m.diaryDir, "diary_jointquestions")
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	case "personal":
		typeDir = filepath.Join(m.diaryDir, "diary_thoughts")
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	default:
		// Для совместимости со старыми записями
		typeDir = m.diaryDir
		filename = filepath.Join(typeDir, fmt.Sprintf("diary_%d.json", userID))
	}
	
	// Создаем папку для типа записи
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки типа записи: %w", err)
	}
	
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

// SaveDiaryEntryWithGender сохраняет запись в дневник с указанием гендера
func (m *Manager) SaveDiaryEntryWithGender(userID int64, username, entry string, week int, entryType, gender string) error {
	diaryEntry := DiaryEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Entry:     entry,
		Week:      week,
		Type:      entryType,
	}

	// Определяем папку и файл в зависимости от типа записи, гендера и недели
	var typeDir, filename string
	switch entryType {
	case "questions":
		typeDir = filepath.Join(m.diaryDir, "diary_questions", fmt.Sprintf("%d", week), gender)
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	case "joint":
		typeDir = filepath.Join(m.diaryDir, "diary_jointquestions", fmt.Sprintf("%d", week), gender)
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	case "personal":
		typeDir = filepath.Join(m.diaryDir, "diary_thoughts", fmt.Sprintf("%d", week), gender)
		filename = filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
	default:
		// Для совместимости со старыми записями
		typeDir = filepath.Join(m.diaryDir, gender)
		filename = filepath.Join(typeDir, fmt.Sprintf("diary_%d.json", userID))
	}
	
	// Создаем папку для типа записи и гендера
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки типа записи: %w", err)
	}
	
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

// ClearUserDiary очищает дневник конкретного пользователя (все типы файлов)
func (m *Manager) ClearUserDiary(userID int64) error {
	// Удаляем файлы пользователя из всех папок типов
	typeDirs := []string{
		"diary_questions",
		"diary_jointquestions", 
		"diary_thoughts",
	}
	
	for _, typeDir := range typeDirs {
		filename := filepath.Join(m.diaryDir, typeDir, fmt.Sprintf("user_%d.json", userID))
		os.Remove(filename) // Игнорируем ошибки если файла нет
	}
	
	// Также удаляем старые файлы для совместимости
	oldFiles := []string{
		fmt.Sprintf("diary_%d.json", userID),
		fmt.Sprintf("diary_questions_%d.json", userID),
		fmt.Sprintf("diary_joint_%d.json", userID),
		fmt.Sprintf("diary_personal_%d.json", userID),
	}
	
	for _, file := range oldFiles {
		filename := filepath.Join(m.diaryDir, file)
		os.Remove(filename) // Игнорируем ошибки для старых файлов
	}
	
	return nil
}


// OpenAIMessage представляет сообщение в формате OpenAI
type OpenAIMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// GetOpenAIHistory возвращает историю в формате OpenAI с ограничением
func (m *Manager) GetOpenAIHistory(userID int64, systemPrompt string, limit int) ([]OpenAIMessage, error) {
	// Загружаем обычную историю
	messages, err := m.GetUserHistory(userID, 0)
	if err != nil {
		return nil, err
	}

	var openaiMessages []OpenAIMessage

	// Добавляем системный промпт в начало
	if systemPrompt != "" {
		openaiMessages = append(openaiMessages, OpenAIMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Берем последние N сообщений (если limit > 0)
	startIdx := 0
	if limit > 0 && len(messages) > limit {
		startIdx = len(messages) - limit
	}

	// Конвертируем в формат OpenAI
	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]
		
		// Добавляем сообщение пользователя
		openaiMessages = append(openaiMessages, OpenAIMessage{
			Role:    "user",
			Content: msg.Message,
		})

		// Добавляем ответ ассистента (очищаем от блоков <think>)
		if msg.Response != "" {
			cleanedResponse := m.cleanResponse(msg.Response)
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "assistant",
				Content: cleanedResponse,
			})
		}
	}

	return openaiMessages, nil
}

// SaveOpenAIMessage сохраняет сообщение в формате совместимом с OpenAI
func (m *Manager) SaveOpenAIMessage(userID int64, username, userMessage, assistantResponse, model string) error {
	return m.SaveMessage(userID, username, userMessage, assistantResponse, model)
}

// cleanResponse очищает ответ от блоков размышлений и лишнего текста
func (m *Manager) cleanResponse(response string) string {
	// Удаляем блоки <think>...</think> (включая многострочные)
	thinkRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")
	
	// Удаляем блоки </think> без открывающего тега (на случай ошибок парсинга)
	thinkEndRegex := regexp.MustCompile(`(?s).*?</think>`)
	cleaned = thinkEndRegex.ReplaceAllString(cleaned, "")
	
	// Удаляем строки, содержащие только </think>
	thinkLineRegex := regexp.MustCompile(`(?m)^.*</think>.*$\n?`)
	cleaned = thinkLineRegex.ReplaceAllString(cleaned, "")
	
	// Удаляем лишние пробелы и переносы строк в начале и конце
	cleaned = strings.TrimSpace(cleaned)
	
	// Удаляем множественные пустые строки
	multipleNewlines := regexp.MustCompile(`\n\s*\n\s*\n`)
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")
	
	// Если после очистки остался пустой ответ, возвращаем исходный
	if cleaned == "" {
		return response
	}
	
	return cleaned
}
