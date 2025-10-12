package scheduling

import (
	"fmt"
	"strings"
	"time"

	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleScheduleTypeCallback обрабатывает выбор типа уведомления и создает задачу
func (h *Handler) HandleScheduleTypeCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Парсим данные: schedule_type_13.10.2025_10:00_diary
	parts := strings.Split(data, "_")
	if len(parts) < 5 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неверный формат данных")
		_, err := h.bot.Send(msg)
		return err
	}

	selectedDate := parts[2] // 13.10.2025
	selectedTime := parts[3] // 10:00
	notificationType := parts[4] // diary/exercise/motivation/custom

	// Парсим дату и время в UTC+5, затем конвертируем в UTC
	utc5 := time.FixedZone("UTC+5", 5*60*60) // 5 часов в секундах

	// Парсим дату и время
	dateTimeStr := fmt.Sprintf("%s %s", selectedDate, selectedTime)
	scheduledTime, err := time.ParseInLocation("02.01.2006 15:04", dateTimeStr, utc5)
	if err != nil {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка парсинга даты/времени")
		_, err := h.bot.Send(msg)
		return err
	}

	// Конвертируем в UTC (отнимаем 5 часов)
	scheduledTimeUTC := scheduledTime.UTC()

	var typeName string
	switch notificationType {
	case "diary":
		typeName = "💌 Мини-дневник"
	case "exercise":
		typeName = "👩🏼‍❤️‍👨🏻 Упражнение недели"
	case "motivation":
		typeName = "💒 Мотивация"
	case "custom":
		// Для кастомных уведомлений нужен отдельный обработчик
		h.userManager.SetState(userID, "schedule_custom_text")
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
			fmt.Sprintf("✏️ Введите текст кастомного уведомления:\n\n"+
				"📅 Дата: %s\n"+
				"🕐 Время: %s (UTC+5)\n\n"+
				"Напишите текст уведомления:", selectedDate, selectedTime))
		_, err := h.bot.Send(msg)
		return err
	default:
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неизвестный тип уведомления")
		_, err := h.bot.Send(msg)
		return err
	}

	// Создаем запланированное уведомление через NotificationService
	var modelType models.NotificationType
	switch notificationType {
	case "diary":
		modelType = models.NotificationDiary
	case "exercise":
		modelType = models.NotificationExercise
	case "motivation":
		modelType = models.NotificationMotivation
	}

	// Планируем уведомление
	scheduleID, err := h.notificationService.ScheduleNotification(scheduledTimeUTC, modelType, nil)
	if err != nil {
		response := fmt.Sprintf("❌ Ошибка планирования уведомления: %v", err)
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err := h.bot.Send(msg)
		return err
	}

	response := fmt.Sprintf("✅ Уведомление запланировано!\n\n"+
		"🆔 ID задачи: %s\n"+
		"📢 Тип: %s\n"+
		"📅 Дата: %s\n"+
		"🕐 Время: %s (UTC+5)\n"+
		"🌍 UTC время: %s\n\n"+
		"Уведомление будет отправлено автоматически в указанное время.",
		scheduleID, typeName, selectedDate, selectedTime, scheduledTimeUTC.Format("02.01.2006 15:04"))

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err = h.bot.Send(msg)
	return err
}

// HandleScheduleCustomTimeCallback обрабатывает кнопку "Свое время"
func (h *Handler) HandleScheduleCustomTimeCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Парсим дату из callback: schedule_custom_time_13.10.2025
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неверный формат данных")
		_, err := h.bot.Send(msg)
		return err
	}

	selectedDate := parts[3] // 13.10.2025

	// Устанавливаем состояние для ввода времени
	h.userManager.SetState(userID, fmt.Sprintf("custom_time_%s", selectedDate))

	text := fmt.Sprintf("⏰ Введите время для %s\n\n"+
		"Формат: ЧЧ:ММ (например: 14:30)\n"+
		"Время указывается в часовом поясе UTC+5", selectedDate)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", fmt.Sprintf("schedule_date_%s", selectedDate)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = kb
	_, err := h.bot.Send(msg)
	return err
}

// HandleScheduleCustomDateCallback обрабатывает кнопку "Своя дата"
func (h *Handler) HandleScheduleCustomDateCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Устанавливаем состояние для ввода даты
	h.userManager.SetState(userID, "custom_date")

	text := "📅 Введите дату\n\n" +
		"Формат: ДД.ММ.ГГГГ (например: 15.10.2025)\n" +
		"Дата должна быть не раньше сегодняшнего дня"

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "schedule_notification"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = kb
	_, err := h.bot.Send(msg)
	return err
}
