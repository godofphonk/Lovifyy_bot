package mocks

import (
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/ai"
	"github.com/godofphonk/lovifyy-bot/internal/history"
	"github.com/godofphonk/lovifyy-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MockAIClient мок для AI клиента
type MockAIClient struct {
	GenerateFunc            func(prompt string) (string, error)
	GenerateWithHistoryFunc func(messages []ai.OpenAIMessage) (string, error)
	TestConnectionFunc      func() error
	SetModelFunc            func(model string)
	GetModelFunc            func() string
}

func (m *MockAIClient) Generate(prompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(prompt)
	}
	return "Mock response", nil
}

func (m *MockAIClient) GenerateWithHistory(messages []ai.OpenAIMessage) (string, error) {
	if m.GenerateWithHistoryFunc != nil {
		return m.GenerateWithHistoryFunc(messages)
	}
	return "Mock response with history", nil
}

func (m *MockAIClient) TestConnection() error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc()
	}
	return nil
}

func (m *MockAIClient) SetModel(model string) {
	if m.SetModelFunc != nil {
		m.SetModelFunc(model)
	}
}

func (m *MockAIClient) GetModel() string {
	if m.GetModelFunc != nil {
		return m.GetModelFunc()
	}
	return "mock-model"
}

// MockTelegramBot мок для Telegram бота
type MockTelegramBot struct {
	SendFunc           func(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChanFunc func(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	RequestFunc        func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	SentMessages       []tgbotapi.Chattable
}

func (m *MockTelegramBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.SentMessages = append(m.SentMessages, c)
	if m.SendFunc != nil {
		return m.SendFunc(c)
	}
	return tgbotapi.Message{MessageID: 1}, nil
}

func (m *MockTelegramBot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	if m.GetUpdatesChanFunc != nil {
		return m.GetUpdatesChanFunc(config)
	}
	return make(tgbotapi.UpdatesChannel)
}

func (m *MockTelegramBot) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	if m.RequestFunc != nil {
		return m.RequestFunc(c)
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

// MockHistoryManager мок для менеджера истории
type MockHistoryManager struct {
	SaveMessageFunc     func(userID int64, username, message, response, model string) error
	GetUserHistoryFunc  func(userID int64, limit int) ([]history.ChatMessage, error)
	GetOpenAIHistoryFunc func(userID int64, systemPrompt string, limit int) ([]history.OpenAIMessage, error)
	SaveDiaryEntryFunc  func(userID int64, username, entry string, week int, entryType string) error
	GetDiaryEntriesFunc func(userID int64, week int, entryType string) ([]history.DiaryEntry, error)
	
	Messages []history.ChatMessage
	Entries  []history.DiaryEntry
}

func (m *MockHistoryManager) SaveMessage(userID int64, username, message, response, model string) error {
	if m.SaveMessageFunc != nil {
		return m.SaveMessageFunc(userID, username, message, response, model)
	}
	
	m.Messages = append(m.Messages, history.ChatMessage{
		UserID:   userID,
		Username: username,
		Message:  message,
		Response: response,
		Model:    model,
	})
	return nil
}

func (m *MockHistoryManager) GetUserHistory(userID int64, limit int) ([]history.ChatMessage, error) {
	if m.GetUserHistoryFunc != nil {
		return m.GetUserHistoryFunc(userID, limit)
	}
	
	var result []history.ChatMessage
	for _, msg := range m.Messages {
		if msg.UserID == userID {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *MockHistoryManager) GetOpenAIHistory(userID int64, systemPrompt string, limit int) ([]history.OpenAIMessage, error) {
	if m.GetOpenAIHistoryFunc != nil {
		return m.GetOpenAIHistoryFunc(userID, systemPrompt, limit)
	}
	
	return []history.OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "Test message"},
	}, nil
}

func (m *MockHistoryManager) SaveDiaryEntry(userID int64, username, entry string, week int, entryType string) error {
	if m.SaveDiaryEntryFunc != nil {
		return m.SaveDiaryEntryFunc(userID, username, entry, week, entryType)
	}
	
	m.Entries = append(m.Entries, history.DiaryEntry{
		UserID:   userID,
		Username: username,
		Entry:    entry,
		Week:     week,
		Type:     entryType,
	})
	return nil
}

func (m *MockHistoryManager) GetDiaryEntries(userID int64, week int, entryType string) ([]history.DiaryEntry, error) {
	if m.GetDiaryEntriesFunc != nil {
		return m.GetDiaryEntriesFunc(userID, week, entryType)
	}
	
	var result []history.DiaryEntry
	for _, entry := range m.Entries {
		if entry.UserID == userID && entry.Week == week && entry.Type == entryType {
			result = append(result, entry)
		}
	}
	return result, nil
}

// MockNotificationService мок для сервиса уведомлений
type MockNotificationService struct {
	GenerateNotificationFunc      func(notificationType models.NotificationType) (string, error)
	SendNotificationToAllFunc     func(message string) error
	SendNotificationToUserFunc    func(userID int64, message string) error
	SendInstantNotificationFunc   func(notificationType models.NotificationType, recipients []int64) error
	GetTemplatesFunc              func() []models.NotificationTemplate
	UpdateTemplateFunc            func(notificationType models.NotificationType, prompt string, isActive bool) error
	AddTemplateFunc               func(template models.NotificationTemplate)
	
	SentNotifications []string
	Recipients        []int64
}

func (m *MockNotificationService) GenerateNotification(notificationType models.NotificationType) (string, error) {
	if m.GenerateNotificationFunc != nil {
		return m.GenerateNotificationFunc(notificationType)
	}
	return "Mock notification: " + string(notificationType), nil
}

func (m *MockNotificationService) SendNotificationToAll(message string) error {
	if m.SendNotificationToAllFunc != nil {
		return m.SendNotificationToAllFunc(message)
	}
	m.SentNotifications = append(m.SentNotifications, message)
	return nil
}

func (m *MockNotificationService) SendNotificationToUser(userID int64, message string) error {
	if m.SendNotificationToUserFunc != nil {
		return m.SendNotificationToUserFunc(userID, message)
	}
	m.SentNotifications = append(m.SentNotifications, message)
	m.Recipients = append(m.Recipients, userID)
	return nil
}

func (m *MockNotificationService) SendInstantNotification(notificationType models.NotificationType, recipients []int64) error {
	if m.SendInstantNotificationFunc != nil {
		return m.SendInstantNotificationFunc(notificationType, recipients)
	}
	message := "Mock notification: " + string(notificationType)
	m.SentNotifications = append(m.SentNotifications, message)
	m.Recipients = append(m.Recipients, recipients...)
	return nil
}

func (m *MockNotificationService) GetTemplates() []models.NotificationTemplate {
	if m.GetTemplatesFunc != nil {
		return m.GetTemplatesFunc()
	}
	return []models.NotificationTemplate{
		{
			Type:      models.NotificationDiary,
			Name:      "Mock Diary Template",
			Prompt:    "Mock prompt",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

func (m *MockNotificationService) UpdateTemplate(notificationType models.NotificationType, prompt string, isActive bool) error {
	if m.UpdateTemplateFunc != nil {
		return m.UpdateTemplateFunc(notificationType, prompt, isActive)
	}
	return nil
}

func (m *MockNotificationService) AddTemplate(template models.NotificationTemplate) {
	if m.AddTemplateFunc != nil {
		m.AddTemplateFunc(template)
	}
}
