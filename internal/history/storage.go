package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// saveToFile сохраняет данные в JSON файл
func (m *Manager) saveToFile(filename string, data interface{}) error {
	// Создаем директорию если не существует
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Сериализуем данные
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// loadFromFile загружает данные из JSON файла
func (m *Manager) loadFromFile(filename string, data interface{}) error {
	// Проверяем существование файла
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // Файл не существует, это нормально
	}

	// Читаем файл
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Десериализуем данные
	if err := json.Unmarshal(fileData, data); err != nil {
		return fmt.Errorf("failed to unmarshal data from %s: %w", filename, err)
	}

	return nil
}

// appendToFile добавляет данные в JSON файл (для массивов)
func (m *Manager) appendToFile(filename string, newItem interface{}) error {
	// Загружаем существующие данные
	var existingData []interface{}
	if err := m.loadFromFile(filename, &existingData); err != nil {
		return err
	}

	// Добавляем новый элемент
	existingData = append(existingData, newItem)

	// Сохраняем обновленные данные
	return m.saveToFile(filename, existingData)
}

// getUserChatFile возвращает путь к файлу чата пользователя
func (m *Manager) getUserChatFile(userID int64) string {
	userDir := filepath.Join(m.historyDir, fmt.Sprintf("user_%d", userID))
	return filepath.Join(userDir, "chat.json")
}

// getUserDiaryFile возвращает путь к файлу дневника пользователя
func (m *Manager) getUserDiaryFile(userID int64) string {
	return filepath.Join(m.diaryDir, fmt.Sprintf("diary_%d.json", userID))
}

// getDiaryTypeFile возвращает путь к файлу дневника по типу
func (m *Manager) getDiaryTypeFile(userID int64, entryType string) string {
	typeDir := filepath.Join(m.diaryDir, fmt.Sprintf("diary_%s", entryType))
	return filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
}

// getDiaryGenderFile возвращает путь к файлу дневника с учетом гендера (старый формат)
func (m *Manager) getDiaryGenderFile(userID int64, entryType, gender string) string {
	typeDir := filepath.Join(m.diaryDir, fmt.Sprintf("diary_%s_%s", entryType, gender))
	return filepath.Join(typeDir, fmt.Sprintf("user_%d.json", userID))
}

// getDiaryStructuredFile возвращает путь к файлу дневника в структурированном формате: gender/week/type/
func (m *Manager) getDiaryStructuredFile(userID int64, gender string, week int, entryType string) string {
	// Создаем структуру: gender/week/type/user_id.json
	structuredDir := filepath.Join(m.diaryDir, gender, fmt.Sprintf("week_%d", week), entryType)
	return filepath.Join(structuredDir, fmt.Sprintf("user_%d.json", userID))
}

// cleanResponse очищает ответ от блоков размышлений и лишнего текста
func (m *Manager) cleanResponse(response string) string {
	// Удаляем блоки <think>...</think> (включая многострочные)
	thinkRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")
	
	// Удаляем лишние пробелы и переносы строк
	cleaned = strings.TrimSpace(cleaned)
	
	// Удаляем множественные переносы строк
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")
	
	return cleaned
}

// fileExists проверяет существование файла
func (m *Manager) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// removeFile удаляет файл
func (m *Manager) removeFile(filename string) error {
	if !m.fileExists(filename) {
		return nil // Файл не существует, это нормально
	}
	return os.Remove(filename)
}
