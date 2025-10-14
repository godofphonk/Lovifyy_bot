package interfaces

import (
	"time"
	"github.com/godofphonk/lovifyy-bot/internal/ai"
	"github.com/godofphonk/lovifyy-bot/internal/history"
	"github.com/godofphonk/lovifyy-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AIClient интерфейс для AI клиентов
type AIClient interface {
	Generate(prompt string) (string, error)
	GenerateWithHistory(messages []ai.OpenAIMessage) (string, error)
	TestConnection() error
	SetModel(model string)
	GetModel() string
}

// TelegramBot интерфейс для Telegram бота
type TelegramBot interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// HistoryManager интерфейс для управления историей
type HistoryManager interface {
	SaveMessage(userID int64, username, message, response, model string) error
	GetUserHistory(userID int64, limit int) ([]history.ChatMessage, error)
	GetOpenAIHistory(userID int64, systemPrompt string, limit int) ([]history.OpenAIMessage, error)
	SaveDiaryEntry(userID int64, username, entry string, week int, entryType string) error
	GetDiaryEntries(userID int64, week int, entryType string) ([]history.DiaryEntry, error)
}

// UserManager интерфейс для управления пользователями
type UserManager interface {
	SetState(userID int64, state string)
	GetState(userID int64) string
	SetStateData(userID int64, state, data string)
	GetStateData(userID int64) (string, string)
	IsAdmin(userID int64) bool
	IsRateLimited(userID int64, limit time.Duration) bool
	ClearState(userID int64)
	GetAdminIDs() []int64
}

// NotificationService интерфейс для уведомлений
type NotificationService interface {
	GenerateNotification(notificationType models.NotificationType) (string, error)
	SendNotificationToAll(message string) error
	SendNotificationToUser(userID int64, message string) error
	SendInstantNotification(notificationType models.NotificationType, recipients []int64) error
	GetTemplates() []models.NotificationTemplate
	UpdateTemplate(notificationType models.NotificationType, prompt string, isActive bool) error
	AddTemplate(template models.NotificationTemplate)
}
