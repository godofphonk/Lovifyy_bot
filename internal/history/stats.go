package history

import (
	"time"
)

// GetStats получает статистику использования чата
func (m *Manager) GetStats(userID int64) (int, time.Time, error) {
	history, err := m.GetUserHistory(userID, 0)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(history) == 0 {
		return 0, time.Time{}, nil
	}

	// Возвращаем количество сообщений и время последнего сообщения
	lastMessage := history[len(history)-1]
	return len(history), lastMessage.Timestamp, nil
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

	// Возвращаем количество записей и время последней записи
	lastEntry := diary[len(diary)-1]
	return len(diary), lastEntry.Timestamp, nil
}

// GetChatStatsDetailed получает детальную статистику чата
func (m *Manager) GetChatStatsDetailed(userID int64) (ChatStats, error) {
	history, err := m.GetUserHistory(userID, 0)
	if err != nil {
		return ChatStats{}, err
	}

	stats := ChatStats{
		TotalMessages: len(history),
		ModelUsage:    make(map[string]int),
	}

	if len(history) == 0 {
		return stats, nil
	}

	// Анализируем историю
	var totalResponseLength int
	for _, msg := range history {
		// Подсчитываем использование моделей
		if msg.Model != "" {
			stats.ModelUsage[msg.Model]++
		}

		// Считаем длину ответов
		totalResponseLength += len(msg.Response)

		// Обновляем временные метки
		if stats.FirstMessage.IsZero() || msg.Timestamp.Before(stats.FirstMessage) {
			stats.FirstMessage = msg.Timestamp
		}
		if msg.Timestamp.After(stats.LastMessage) {
			stats.LastMessage = msg.Timestamp
		}
	}

	// Вычисляем среднюю длину ответа
	if len(history) > 0 {
		stats.AverageResponseLength = totalResponseLength / len(history)
	}

	return stats, nil
}

// GetDiaryStatsDetailed получает детальную статистику дневника
func (m *Manager) GetDiaryStatsDetailed(userID int64) (DiaryStats, error) {
	diary, err := m.GetUserDiary(userID, 0)
	if err != nil {
		return DiaryStats{}, err
	}

	stats := DiaryStats{
		TotalEntries: len(diary),
		WeekStats:    make(map[int]int),
		TypeStats:    make(map[string]int),
	}

	if len(diary) == 0 {
		return stats, nil
	}

	// Анализируем записи дневника
	var totalEntryLength int
	for _, entry := range diary {
		// Подсчитываем записи по неделям
		stats.WeekStats[entry.Week]++

		// Подсчитываем записи по типам
		if entry.Type != "" {
			stats.TypeStats[entry.Type]++
		}

		// Считаем длину записей
		totalEntryLength += len(entry.Entry)

		// Обновляем временные метки
		if stats.FirstEntry.IsZero() || entry.Timestamp.Before(stats.FirstEntry) {
			stats.FirstEntry = entry.Timestamp
		}
		if entry.Timestamp.After(stats.LastEntry) {
			stats.LastEntry = entry.Timestamp
		}
	}

	// Вычисляем среднюю длину записи
	if len(diary) > 0 {
		stats.AverageEntryLength = totalEntryLength / len(diary)
	}

	return stats, nil
}

// ChatStats представляет детальную статистику чата
type ChatStats struct {
	TotalMessages         int            `json:"total_messages"`
	FirstMessage          time.Time      `json:"first_message"`
	LastMessage           time.Time      `json:"last_message"`
	AverageResponseLength int            `json:"average_response_length"`
	ModelUsage            map[string]int `json:"model_usage"`
}

// DiaryStats представляет детальную статистику дневника
type DiaryStats struct {
	TotalEntries       int            `json:"total_entries"`
	FirstEntry         time.Time      `json:"first_entry"`
	LastEntry          time.Time      `json:"last_entry"`
	AverageEntryLength int            `json:"average_entry_length"`
	WeekStats          map[int]int    `json:"week_stats"`
	TypeStats          map[string]int `json:"type_stats"`
}
