package history

import (
	"fmt"
	"path/filepath"
	"time"
)

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

	filename := m.getDiaryTypeFile(userID, entryType)
	
	// Загружаем существующие записи
	var entries []DiaryEntry
	if err := m.loadFromFile(filename, &entries); err != nil {
		return fmt.Errorf("failed to load existing diary entries: %w", err)
	}

	// Добавляем новую запись
	entries = append(entries, diaryEntry)

	// Сохраняем обновленные записи
	return m.saveToFile(filename, entries)
}

// SaveDiaryEntryWithGender сохраняет запись в дневник с указанием гендера (новый структурированный формат)
func (m *Manager) SaveDiaryEntryWithGender(userID int64, username, entry string, week int, entryType, gender string) error {
	diaryEntry := DiaryEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Entry:     entry,
		Week:      week,
		Type:      entryType,
	}

	// Используем новый структурированный формат: gender/week/type/
	filename := m.getDiaryStructuredFile(userID, gender, week, entryType)
	
	// Загружаем существующие записи
	var entries []DiaryEntry
	if err := m.loadFromFile(filename, &entries); err != nil {
		return fmt.Errorf("failed to load existing diary entries: %w", err)
	}

	// Добавляем новую запись
	entries = append(entries, diaryEntry)

	// Сохраняем обновленные записи
	return m.saveToFile(filename, entries)
}

// GetDiaryEntriesByWeek получает записи дневника для конкретной недели (только questions и personal)
func (m *Manager) GetDiaryEntriesByWeek(userID int64, week int) ([]DiaryEntry, error) {
	var allWeekEntries []DiaryEntry
	
	// Читаем записи из папки "diary_questions"
	questionsFile := m.getDiaryTypeFile(userID, "questions")
	var questionsEntries []DiaryEntry
	if err := m.loadFromFile(questionsFile, &questionsEntries); err == nil {
		for _, entry := range questionsEntries {
			if entry.Week == week {
				allWeekEntries = append(allWeekEntries, entry)
			}
		}
	}
	
	// Читаем записи из папки "diary_personal"
	personalFile := m.getDiaryTypeFile(userID, "personal")
	var personalEntries []DiaryEntry
	if err := m.loadFromFile(personalFile, &personalEntries); err == nil {
		for _, entry := range personalEntries {
			if entry.Week == week {
				allWeekEntries = append(allWeekEntries, entry)
			}
		}
	}
	
	return allWeekEntries, nil
}

// GetUserDiary получает записи дневника пользователя
func (m *Manager) GetUserDiary(userID int64, limit int) ([]DiaryEntry, error) {
	filename := m.getUserDiaryFile(userID)
	
	var diary []DiaryEntry
	if err := m.loadFromFile(filename, &diary); err != nil {
		return nil, fmt.Errorf("failed to load diary: %w", err)
	}

	// Применяем лимит если указан
	if limit > 0 && len(diary) > limit {
		diary = diary[len(diary)-limit:]
	}

	return diary, nil
}

// ClearUserDiary очищает дневник конкретного пользователя (все типы файлов)
func (m *Manager) ClearUserDiary(userID int64) error {
	// Удаляем файлы пользователя из всех папок типов
	typeDirs := []string{
		"diary_questions",
		"diary_personal", 
		"diary_joint",
		"diary_questions_male",
		"diary_questions_female",
		"diary_personal_male",
		"diary_personal_female",
	}

	var lastError error
	for _, typeDir := range typeDirs {
		filename := filepath.Join(m.diaryDir, typeDir, fmt.Sprintf("user_%d.json", userID))
		if err := m.removeFile(filename); err != nil {
			lastError = err // Сохраняем последнюю ошибку, но продолжаем удаление
		}
	}

	// Также удаляем основной файл дневника
	mainFile := m.getUserDiaryFile(userID)
	if err := m.removeFile(mainFile); err != nil {
		lastError = err
	}

	return lastError
}

// GetDiaryEntriesByType получает записи дневника по типу
func (m *Manager) GetDiaryEntriesByType(userID int64, entryType string) ([]DiaryEntry, error) {
	filename := m.getDiaryTypeFile(userID, entryType)
	
	var entries []DiaryEntry
	if err := m.loadFromFile(filename, &entries); err != nil {
		return nil, fmt.Errorf("failed to load diary entries for type %s: %w", entryType, err)
	}

	return entries, nil
}

// GetDiaryEntriesByTypeAndGender получает записи дневника по типу и гендеру (старый формат)
func (m *Manager) GetDiaryEntriesByTypeAndGender(userID int64, entryType, gender string) ([]DiaryEntry, error) {
	filename := m.getDiaryGenderFile(userID, entryType, gender)
	
	var entries []DiaryEntry
	if err := m.loadFromFile(filename, &entries); err != nil {
		return nil, fmt.Errorf("failed to load diary entries for type %s and gender %s: %w", entryType, gender, err)
	}

	return entries, nil
}

// GetDiaryEntriesStructured получает записи дневника по структурированному формату: gender/week/type
func (m *Manager) GetDiaryEntriesStructured(userID int64, gender string, week int, entryType string) ([]DiaryEntry, error) {
	filename := m.getDiaryStructuredFile(userID, gender, week, entryType)
	
	var entries []DiaryEntry
	if err := m.loadFromFile(filename, &entries); err != nil {
		return nil, fmt.Errorf("failed to load diary entries: %w", err)
	}
	
	return entries, nil
}

// GetAllDiaryEntriesForWeekAndGender получает ВСЕ записи дневника для конкретной недели и гендера (все типы)
func (m *Manager) GetAllDiaryEntriesForWeekAndGender(userID int64, gender string, week int) ([]DiaryEntry, error) {
	var allEntries []DiaryEntry
	
	// Получаем записи всех типов для данной недели и гендера
	types := []string{"personal", "questions", "joint"}
	
	for _, entryType := range types {
		entries, err := m.GetDiaryEntriesStructured(userID, gender, week, entryType)
		if err != nil {
			// Если файл не существует, это нормально - просто пропускаем
			continue
		}
		allEntries = append(allEntries, entries...)
	}
	
	return allEntries, nil
}
