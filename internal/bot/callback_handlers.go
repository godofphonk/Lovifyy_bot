package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCallbackQuery обрабатывает callback queries
func (b *EnterpriseBot) handleCallbackQuery(update tgbotapi.Update) error {
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := b.telegram.Request(callback); err != nil {
		b.logger.WithError(err).Error("Failed to answer callback query")
	}

	data := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID

	// Регистрируем пользователя в системе уведомлений
	b.notificationService.RegisterUser(userID, update.CallbackQuery.From.UserName)
	
	// Обновляем активность пользователя
	b.notificationService.UpdateUserActivity(userID)

	b.logger.WithFields(map[string]interface{}{
		"user_id":       userID,
		"callback_data": data,
	}).Info("Processing callback query")

    // Роутинг callback queries как в legacy
    switch {
    case data == "chat":
        return b.commandHandler.HandleCallback(update)
    case data == "advice":
        return b.commandHandler.HandleCallback(update)
    case data == "diary":
        return b.commandHandler.HandleCallback(update)
    case data == "adminhelp":
        return b.commandHandler.HandleCallback(update)
    case data == "notifications_menu":
        return b.commandHandler.HandleCallback(update)
    case data == "schedule_notification":
        return b.commandHandler.HandleCallback(update)
    case data == "view_notifications":
        return b.commandHandler.HandleCallback(update)
    case data == "send_now":
        return b.commandHandler.HandleCallback(update)
    case data == "notify_custom":
        return b.commandHandler.HandleCallback(update)
    case data == "notify_schedule_custom":
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "schedule_date_"):
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "schedule_time_"):
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "schedule_type_"):
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "schedule_custom_time_"):
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "schedule_custom_date"):
        return b.commandHandler.HandleCallback(update)
    case data == "mode_chat":
        return b.handleChatMode(userID)
    case data == "mode_diary":
        return b.handleDiaryMode(userID)
    case data == "exercises":
        // Делегируем показ недель в CommandHandler (как в legacy логике меню)
        return b.commandHandler.HandleCallback(update)
    case data == "exercise_week_1":
        return b.handleExerciseWeekCallbackNew(userID, 1)
    case data == "exercise_week_2":
        return b.handleExerciseWeekCallbackNew(userID, 2)
    case data == "exercise_week_3":
        return b.handleExerciseWeekCallbackNew(userID, 3)
    case data == "exercise_week_4":
        return b.handleExerciseWeekCallbackNew(userID, 4)
    case strings.HasPrefix(data, "diary_type_"):
        return b.handleDiaryTypeCallback(userID, data)
    case strings.HasPrefix(data, "diary_gender_"):
        return b.handleDiaryGenderCallback(userID, data)
    case strings.HasPrefix(data, "week_"):
        // Делегируем обработку недель в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "notify_send_all_"):
        // Делегируем обработку уведомлений в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "notify_"):
        // Делегируем обработку уведомлений в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "final_insight") || data == "generate_final_insight":
        // Делегируем обработку финального инсайта в CommandHandler
        return b.commandHandler.HandleCallback(update)
    default:
        b.logger.WithField("callback_data", data).Warn("Unknown callback query")
        return nil
    }
}

// handleExerciseWeekCallbackNew обрабатывает выбор недели упражнений (новая версия)
func (b *EnterpriseBot) handleExerciseWeekCallbackNew(userID int64, week int) error {
    // Делегируем в CommandHandler для совместимости
    msg := tgbotapi.NewMessage(userID, "📅 Функция недель упражнений в разработке")
    _, err := b.telegram.Send(msg)
    return err
}

// handleDiaryGenderCallback обрабатывает выбор пола для дневника
func (b *EnterpriseBot) handleDiaryGenderCallback(userID int64, data string) error {
    // Парсим callback data: diary_gender_male или diary_gender_female
    parts := strings.Split(data, "_")
    if len(parts) < 3 {
        msg := tgbotapi.NewMessage(userID, "❌ Неверный формат данных")
        _, err := b.telegram.Send(msg)
        return err
    }

    gender := parts[2] // male или female

    // Показываем типы записей
    text := "📔 Выберите тип записи в дневнике:"
    
    var genderEmoji string
    if gender == "male" {
        genderEmoji = "👨"
    } else {
        genderEmoji = "👩"
    }

    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("📝 Личные размышления", "diary_type_personal_"+gender),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("❓ Ответы на вопросы", "diary_type_questions_"+gender),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "mode_diary"),
        ),
    )

    msg := tgbotapi.NewMessage(userID, genderEmoji+" "+text)
    msg.ReplyMarkup = keyboard
    _, err := b.telegram.Send(msg)
    return err
}

