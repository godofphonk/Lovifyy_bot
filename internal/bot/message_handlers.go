package bot

import (
	"fmt"
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
		// Проверяем, не является ли это состоянием дневника
		if strings.HasPrefix(state, "diary_") {
			return b.handleDiaryMessage(userID, sanitizedText)
		}
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
	state := b.userManager.GetState(userID)
	
	// Парсим состояние: diary_<gender>_<week>_<type>
	if strings.HasPrefix(state, "diary_") {
		parts := strings.Split(state, "_")
		if len(parts) >= 4 {
			gender := parts[1]
			week := parts[2]
			diaryType := parts[3]
			
			// Конвертируем неделю в число
			weekNum := 1
			switch week {
			case "1":
				weekNum = 1
			case "2":
				weekNum = 2
			case "3":
				weekNum = 3
			case "4":
				weekNum = 4
			}
			
			// Сохраняем запись в дневник с полной информацией
			err := b.historyManager.SaveDiaryEntryWithGender(userID, "user", messageText, weekNum, diaryType, gender)
			if err != nil {
				b.logger.WithError(err).Error("Failed to save diary entry")
				msg := tgbotapi.NewMessage(userID, "❌ Ошибка при сохранении записи")
				_, err := b.telegram.Send(msg)
				return err
			}
			
			// Определяем эмодзи и текст для ответа
			var genderEmoji string
			var typeEmoji string
			var typeText string
			
			if gender == "male" {
				genderEmoji = "👨"
			} else {
				genderEmoji = "👩"
			}
			
			switch diaryType {
			case "personal":
				typeEmoji = "💭"
				typeText = "Личные мысли"
			case "questions":
				typeEmoji = "❓"
				typeText = "Ответы на вопросы"
			case "joint":
				typeEmoji = "👫"
				typeText = "Ответы на совместные вопросы"
			default:
				typeEmoji = "📝"
				typeText = "Запись"
			}
			
			response := fmt.Sprintf("✅ Запись сохранена!\n\n"+
				"%s %s - Неделя %s\n"+
				"%s %s\n\n"+
				"📝 Продолжайте писать или используйте /start для возврата в главное меню.", 
				genderEmoji, 
				map[string]string{"male": "Парень", "female": "Девушка"}[gender], 
				week, typeEmoji, typeText)
			
			msg := tgbotapi.NewMessage(userID, response)
			_, err = b.telegram.Send(msg)
			return err
		}
	}
	
	// Если состояние просто "diary" (старый формат)
	if state == "diary" {
		// Сохраняем как общую запись
		err := b.historyManager.SaveDiaryEntry(userID, "user", messageText, 1, "general")
		if err != nil {
			b.logger.WithError(err).Error("Failed to save diary entry")
			msg := tgbotapi.NewMessage(userID, "❌ Ошибка при сохранении записи")
			_, err := b.telegram.Send(msg)
			return err
		}
		
		msg := tgbotapi.NewMessage(userID, "📝 Записано! Продолжайте писать или используйте /start для возврата в главное меню.")
		_, err = b.telegram.Send(msg)
		return err
	}
	
	// Неизвестное состояние дневника
	msg := tgbotapi.NewMessage(userID, "❓ Неизвестное состояние дневника. Используйте /start для возврата в главное меню.")
	_, err := b.telegram.Send(msg)
	return err
}
