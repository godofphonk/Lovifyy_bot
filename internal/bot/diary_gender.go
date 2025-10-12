package bot

import (
	"fmt"
	"strings"
	"time"
	
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleDiaryTypeCallback обрабатывает выбор типа записи в дневнике
func (b *EnterpriseBot) handleDiaryTypeCallback(userID int64, data string) error {
	// Парсим callback data: diary_type_personal_male или diary_type_questions_female
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return fmt.Errorf("invalid diary type callback data: %s", data)
	}
	
	entryType := parts[2] // personal, questions, joint
	gender := parts[3]    // male, female
	
	// Устанавливаем состояние для записи в дневник
	state := fmt.Sprintf("diary_entry_%s_%s", entryType, gender)
	b.userManager.SetState(userID, state)
	
	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}
	
	var typeDescription string
	switch entryType {
	case "personal":
		typeDescription = "личные мысли и переживания"
	case "questions":
		typeDescription = "ответы на упражнения недели"
	case "joint":
		typeDescription = "совместные размышления"
	}
	
	text := fmt.Sprintf("%s **Дневник для %s**\n\n📝 Напишите %s.\n\nПросто отправьте сообщение, и я сохраню его в дневнике.", 
		genderEmoji, genderName, typeDescription)
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к типам", fmt.Sprintf("diary_gender_%s", gender)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)
	
	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, err := b.telegram.Send(msg)
	return err
}

// handleDiaryMessage обрабатывает сообщения в режиме дневника с гендером
func (b *EnterpriseBot) handleDiaryMessageWithGender(userID int64, messageText string, state string) error {
	// Парсим состояние: diary_entry_personal_male
	parts := strings.Split(state, "_")
	if len(parts) < 4 {
		return fmt.Errorf("invalid diary state: %s", state)
	}
	
	entryType := parts[2] // personal, questions, joint
	gender := parts[3]    // male, female
	
	// Сохраняем запись в дневник с гендером
	err := b.saveDiaryEntryWithGender(userID, entryType, gender, messageText)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, "❌ Ошибка сохранения записи в дневник")
		b.telegram.Send(msg)
		return err
	}
	
	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}
	
	text := fmt.Sprintf("✅ Запись сохранена в дневник для %s!\n\n%s Можете продолжить писать или выбрать другой тип записи.", genderName, genderEmoji)
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Продолжить писать", fmt.Sprintf("diary_type_%s_%s", entryType, gender)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Другой тип записи", fmt.Sprintf("diary_gender_%s", gender)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)
	
	msg := tgbotapi.NewMessage(userID, text)
	msg.ReplyMarkup = keyboard
	_, err = b.telegram.Send(msg)
	return err
}

// saveDiaryEntryWithGender сохраняет запись в дневник с учетом гендера
func (b *EnterpriseBot) saveDiaryEntryWithGender(userID int64, entryType, gender, content string) error {
	// Определяем текущую неделю (можно улучшить логику)
	currentWeek := 1 // Пока используем неделю 1, можно добавить логику определения недели
	
	// Создаем структуру записи
	entry := struct {
		UserID    int64  `json:"user_id"`
		Week      int    `json:"week"`
		Type      string `json:"type"`
		Gender    string `json:"gender"`
		Content   string `json:"content"`
		Timestamp string `json:"timestamp"`
	}{
		UserID:    userID,
		Week:      currentWeek,
		Type:      entryType,
		Gender:    gender,
		Content:   content,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
	}
	
	// Здесь должна быть логика сохранения в файл или базу данных
	// Пока просто логируем с данными entry
	b.logger.WithFields(map[string]interface{}{
		"user_id":    entry.UserID,
		"entry_type": entry.Type,
		"gender":     entry.Gender,
		"week":       entry.Week,
		"content":    entry.Content,
	}).Info("Diary entry saved with gender")
	
	return nil
}
