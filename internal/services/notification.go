package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// NotificationService управляет уведомлениями
type NotificationService struct {
	bot         *tgbotapi.BotAPI
	ai          *ai.OpenAIClient
	templates   []models.NotificationTemplate
	dataDir     string
	userStorage *models.UserStorage
}

// NewNotificationService создает новый сервис уведомлений
func NewNotificationService(bot *tgbotapi.BotAPI, ai *ai.OpenAIClient) *NotificationService {
	dataDir := "data/notifications"
	service := &NotificationService{
		bot:         bot,
		ai:          ai,
		templates:   models.GetDefaultTemplates(),
		dataDir:     dataDir,
		userStorage: models.NewUserStorage("data"),
	}
	
	// Создаем директорию для данных
	os.MkdirAll(service.dataDir, 0755)
	
	// Загружаем шаблоны из файла
	service.loadTemplates()
	
	return service
}

// loadTemplates загружает шаблоны из файла
func (ns *NotificationService) loadTemplates() {
	filePath := filepath.Join(ns.dataDir, "templates.json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Если файл не существует, сохраняем стандартные шаблоны
		ns.saveTemplates()
		return
	}
	
	var templates []models.NotificationTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		log.Printf("❌ Ошибка загрузки шаблонов: %v", err)
		return
	}
	
	ns.templates = templates
}

// saveTemplates сохраняет шаблоны в файл
func (ns *NotificationService) saveTemplates() {
	filePath := filepath.Join(ns.dataDir, "templates.json")
	
	data, err := json.MarshalIndent(ns.templates, "", "  ")
	if err != nil {
		log.Printf("❌ Ошибка сериализации шаблонов: %v", err)
		return
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("❌ Ошибка сохранения шаблонов: %v", err)
	}
}

// GenerateNotification генерирует уведомление с помощью AI
func (ns *NotificationService) GenerateNotification(notificationType models.NotificationType) (string, error) {
	// Находим шаблон для данного типа
	var template *models.NotificationTemplate
	for i := range ns.templates {
		if ns.templates[i].Type == notificationType && ns.templates[i].IsActive {
			template = &ns.templates[i]
			break
		}
	}
	
	if template == nil {
		return "", fmt.Errorf("шаблон для типа %s не найден или неактивен", notificationType)
	}
	
	// Генерируем сообщение с помощью AI
	response, err := ns.ai.Generate(template.Prompt)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации уведомления: %v", err)
	}
	
	return response, nil
}

// SendNotificationToAll отправляет уведомление всем пользователям
func (ns *NotificationService) SendNotificationToAll(message string) error {
	log.Printf("📢 Отправка уведомления всем пользователям: %s", message)
	
	// Получаем список всех активных пользователей из JSON файла
	userIDs, err := ns.userStorage.GetUserIDs()
	if err != nil {
		log.Printf("❌ Ошибка получения списка пользователей: %v", err)
		// Fallback: отправляем только админам
		userIDs = []int64{1805441944} // ID админа
	}
	
	if len(userIDs) == 0 {
		log.Printf("⚠️ Нет активных пользователей для отправки уведомлений")
		return nil
	}
	
	successCount := 0
	errorCount := 0
	
	for _, userID := range userIDs {
		msg := tgbotapi.NewMessage(userID, message)
		msg.ParseMode = "HTML"
		
		_, err := ns.bot.Send(msg)
		if err != nil {
			log.Printf("❌ Ошибка отправки уведомления пользователю %d: %v", userID, err)
			errorCount++
			continue
		}
		log.Printf("✅ Уведомление отправлено пользователю %d", userID)
		successCount++
	}
	
	log.Printf("📊 Статистика отправки: успешно %d, ошибок %d, всего %d", successCount, errorCount, len(userIDs))
	return nil
}

// SendCustomNotification отправляет кастомное уведомление всем пользователям
func (ns *NotificationService) SendCustomNotification(message string) error {
	// Используем существующий метод для отправки всем
	return ns.SendNotificationToAll(message)
}

// SendNotificationToUser отправляет уведомление конкретному пользователю
func (ns *NotificationService) SendNotificationToUser(userID int64, message string) error {
	msg := tgbotapi.NewMessage(userID, message)
	msg.ParseMode = "HTML"
	
	_, err := ns.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки уведомления пользователю %d: %v", userID, err)
	}
	
	log.Printf("📤 Уведомление отправлено пользователю %d", userID)
	return nil
}

// SendInstantNotification отправляет мгновенное уведомление
func (ns *NotificationService) SendInstantNotification(notificationType models.NotificationType, recipients []int64) error {
	// Генерируем сообщение
	message, err := ns.GenerateNotification(notificationType)
	if err != nil {
		return fmt.Errorf("ошибка генерации уведомления: %v", err)
	}
	
	// Если получатели не указаны, отправляем всем
	if len(recipients) == 0 {
		return ns.SendNotificationToAll(message)
	}
	
	// Отправляем конкретным пользователям
	for _, userID := range recipients {
		if err := ns.SendNotificationToUser(userID, message); err != nil {
			log.Printf("❌ Ошибка отправки уведомления пользователю %d: %v", userID, err)
		}
	}
	
	return nil
}

// GetTemplates возвращает все шаблоны
func (ns *NotificationService) GetTemplates() []models.NotificationTemplate {
	return ns.templates
}

// UpdateTemplate обновляет шаблон
func (ns *NotificationService) UpdateTemplate(notificationType models.NotificationType, prompt string, isActive bool) error {
	for i := range ns.templates {
		if ns.templates[i].Type == notificationType {
			ns.templates[i].Prompt = prompt
			ns.templates[i].IsActive = isActive
			ns.templates[i].UpdatedAt = time.Now()
			ns.saveTemplates()
			return nil
		}
	}
	
	return fmt.Errorf("шаблон типа %s не найден", notificationType)
}

// AddTemplate добавляет новый шаблон
func (ns *NotificationService) AddTemplate(template models.NotificationTemplate) {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	ns.templates = append(ns.templates, template)
	ns.saveTemplates()
}

// RegisterUser регистрирует пользователя в системе уведомлений
func (ns *NotificationService) RegisterUser(userID int64, username string) error {
	return ns.userStorage.AddUser(userID, username)
}

// UpdateUserActivity обновляет активность пользователя
func (ns *NotificationService) UpdateUserActivity(userID int64) error {
	return ns.userStorage.UpdateLastSeen(userID)
}

// GetUserCount возвращает количество активных пользователей
func (ns *NotificationService) GetUserCount() (int, error) {
	return ns.userStorage.GetUserCount()
}
