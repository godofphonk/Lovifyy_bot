package models

import "time"

// NotificationType представляет тип уведомления
type NotificationType string

const (
	NotificationDiary      NotificationType = "diary"
	NotificationExercise   NotificationType = "exercise"
	NotificationMotivation NotificationType = "motivation"
	NotificationCustom     NotificationType = "custom"
)

// Notification представляет уведомление
type Notification struct {
	ID          string           `json:"id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Message     string           `json:"message"`
	ScheduledAt time.Time        `json:"scheduled_at"`
	CreatedAt   time.Time        `json:"created_at"`
	SentAt      *time.Time       `json:"sent_at,omitempty"`
	IsActive    bool             `json:"is_active"`
	Recipients  []int64          `json:"recipients,omitempty"` // Если пусто - всем пользователям
}

// NotificationTemplate представляет шаблон для динамической генерации
type NotificationTemplate struct {
	Type        NotificationType `json:"type"`
	Name        string           `json:"name"`
	Prompt      string           `json:"prompt"`       // Промпт для GPT генерации
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// GetDefaultTemplates возвращает стандартные шаблоны уведомлений
func GetDefaultTemplates() []NotificationTemplate {
	return []NotificationTemplate{
		{
			Type:      NotificationDiary,
			Name:      "Напоминание о дневнике",
			Prompt:    "Создай теплое и мотивирующее напоминание парам о ведении дневника отношений. Сообщение должно быть на русском языке, дружелюбным и вдохновляющим. Используй эмодзи. Длина 50-100 слов.",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Type:      NotificationExercise,
			Name:      "Напоминание об упражнениях",
			Prompt:    "Создай мотивирующее сообщение парам о выполнении ПСИХОЛОГИЧЕСКИХ упражнений для укрепления отношений и эмоциональной близости. НЕ физические упражнения! Речь идет о психологических практиках, упражнениях на доверие, общение, взаимопонимание между партнерами. Сообщение должно быть на русском языке, позитивным и вдохновляющим. Используй эмодзи. Длина 50-100 слов.",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Type:      NotificationMotivation,
			Name:      "Мотивационное сообщение",
			Prompt:    "Создай вдохновляющее сообщение для пар о важности работы над отношениями. Сообщение должно быть на русском языке, теплым, поддерживающим и мотивирующим. Используй эмодзи. Длина 50-100 слов.",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}
