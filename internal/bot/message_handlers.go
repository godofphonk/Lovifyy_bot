package bot

import (
	"strings"
	"time"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/history"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleMessage обрабатывает обычные сообщения с валидацией
func (b *EnterpriseBot) handleMessage(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	messageText := update.Message.Text
	
	// Регистрируем пользователя в системе уведомлений
	b.notificationService.RegisterUser(userID, update.Message.From.UserName)
	
	// Обновляем активность пользователя
	b.notificationService.UpdateUserActivity(userID)

	// Валидируем сообщение
	if validation := b.validator.ValidateMessage(messageText); !validation.Valid {
		b.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"errors":  validation.Errors,
		}).Warn("Invalid message received")
		
		if b.metrics != nil {
			b.metrics.RecordError("invalid_message", "validation")
		}
		
		// Отправляем сообщение об ошибке
		errorMsg := "❌ Сообщение содержит недопустимый контент"
		msg := tgbotapi.NewMessage(userID, errorMsg)
		b.telegram.Send(msg)
		return nil
	}

	// Санитизируем сообщение
	sanitizedText := b.validator.SanitizeMessage(messageText)

	// Записываем метрики
	if b.metrics != nil {
		b.metrics.RecordMessage("text", "received")
		b.metrics.RecordMessageLength("user", float64(len(sanitizedText)))
	}

	// Получаем состояние пользователя
	state := b.userManager.GetState(userID)

	switch state {
	case "chat":
		return b.handleChatMessage(userID, sanitizedText)
	case "diary":
		return b.handleDiaryMessage(userID, sanitizedText)
	case "custom_notification":
		return b.handleCustomNotificationMessage(userID, sanitizedText)
	case "custom_notification_schedule":
		return b.handleCustomNotificationScheduleMessage(userID, sanitizedText)
	case "schedule_custom_text":
		return b.handleScheduleCustomTextMessage(userID, sanitizedText)
	default:
		// Проверяем, не является ли это состоянием кастомного времени
		if strings.HasPrefix(state, "custom_time_") {
			return b.handleCustomTimeMessage(userID, sanitizedText, state)
		}
		if state == "custom_date" {
			return b.handleCustomDateMessage(userID, sanitizedText)
		}
		return b.suggestMode(userID)
	}
}

// handleChatMessage обрабатывает сообщения в режиме чата
func (b *EnterpriseBot) handleChatMessage(userID int64, messageText string) error {
	startTime := time.Now()
	
	// Проверяем доступность AI
	if b.ai == nil {
		msg := tgbotapi.NewMessage(userID, "❌ AI сервис временно недоступен")
		_, err := b.telegram.Send(msg)
		return err
	}

	// Получаем историю с системным промптом
	historyMessages, err := b.historyManager.GetOpenAIHistory(userID, b.config.Telegram.SystemPrompt, 10)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get chat history")
		// Продолжаем без истории - создаем только системный промпт
		historyMessages = []history.OpenAIMessage{
			{
				Role:    "system",
				Content: b.config.Telegram.SystemPrompt,
			},
		}
	}
	
	// Добавляем текущее сообщение пользователя
	historyMessages = append(historyMessages, history.OpenAIMessage{
		Role:    "user",
		Content: messageText,
	})

	// Конвертируем в формат ai пакета
	aiMessages := make([]ai.OpenAIMessage, len(historyMessages))
	for i, msg := range historyMessages {
		aiMessages[i] = ai.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Генерируем ответ с историей и системным промптом
	response, err := b.ai.GenerateWithHistory(aiMessages)
	if err != nil {
		b.logger.WithError(err).Error("Failed to generate AI response")
		
		if b.metrics != nil {
			b.metrics.RecordError("ai_generation", "openai")
		}
		
		msg := tgbotapi.NewMessage(userID, "❌ Извините, произошла ошибка при генерации ответа")
		_, err := b.telegram.Send(msg)
		return err
	}

	// Сохраняем в историю
	if err := b.historyManager.SaveMessage(userID, messageText, response, "chat", "user"); err != nil {
		b.logger.WithError(err).Error("Failed to save message to history")
	}

	// Отправляем ответ
	msg := tgbotapi.NewMessage(userID, response)
	_, err = b.telegram.Send(msg)

	// Записываем метрики
	if b.metrics != nil {
		duration := time.Since(startTime)
		b.metrics.RecordResponseDuration("ai_chat", "openai", duration)
		b.metrics.RecordMessage("text", "sent")
		b.metrics.RecordMessageLength("bot", float64(len(response)))
	}

	return err
}

// handleDiaryMessage обрабатывает сообщения в режиме дневника
func (b *EnterpriseBot) handleDiaryMessage(userID int64, messageText string) error {
	// TODO: Implement diary message handling
	msg := tgbotapi.NewMessage(userID, "📔 Функция дневника в разработке")
	_, err := b.telegram.Send(msg)
	return err
}
