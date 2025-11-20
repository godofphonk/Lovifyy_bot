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
    case data == "show_recipients":
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
        // Делегируем обработку типов записи дневника в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "diary_gender_"):
        // Делегируем обработку выбора пола дневника в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "diary_week_"):
        // Делегируем обработку выбора недели дневника в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case data == "diary_view":
        // Делегируем обработку просмотра записей дневника в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case strings.HasPrefix(data, "diary_view_"):
        // Делегируем обработку просмотра записей дневника в CommandHandler
        return b.commandHandler.HandleCallback(update)
    case data == "main_menu":
        // Делегируем обработку главного меню в CommandHandler
        return b.commandHandler.HandleCallback(update)
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


