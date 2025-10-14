package scheduling

import (
	"fmt"
	"strings"
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/models"
	"github.com/godofphonk/lovifyy-bot/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает планирование уведомлений
type Handler struct {
	bot                 *tgbotapi.BotAPI
	userManager         *models.UserManager
	notificationService *services.NotificationService
}

// NewHandler создает новый обработчик планирования
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, notificationService *services.NotificationService) *Handler {
	return &Handler{
		bot:                 bot,
		userManager:         userManager,
		notificationService: notificationService,
	}
}

// HandleScheduleNotification обрабатывает планирование уведомлений как в legacy
func (h *Handler) HandleScheduleNotification(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "⏰ Запланировать уведомление\n\n" +
		"🗓️ Выберите дату отправки:\n" +
		"🕐 Часовой пояс: UTC+5 (Алматы/Ташкент)"

	// Создаем кнопки с датами (сегодня + следующие 6 дней) как в legacy
	var buttons [][]tgbotapi.InlineKeyboardButton

	// Получаем текущее время в UTC+5 (фиксированный offset)
	utc5 := time.FixedZone("UTC+5", 5*60*60) // 5 часов в секундах
	nowUTC5 := time.Now().In(utc5)

	for i := 0; i < 7; i++ {
		date := nowUTC5.AddDate(0, 0, i)
		dateStr := date.Format("02.01.2006")
		var dayName string

		switch i {
		case 0:
			dayName = "Сегодня"
		case 1:
			dayName = "Завтра"
		default:
			dayName = date.Format("Mon") // Wed, Thu, Fri, Sat, Sun
		}

		buttonText := fmt.Sprintf("%s (%s)", dayName, dateStr)
		callbackData := fmt.Sprintf("schedule_date_%s", dateStr)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
		))
	}

	// Добавляем кнопку "Своя дата"
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📅 Своя дата", "schedule_custom_date"),
	))

	// Кнопка назад
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "notifications_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleScheduleDateCallback обрабатывает выбор даты для планирования
func (h *Handler) HandleScheduleDateCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Парсим дату из callback: schedule_date_13.10.2025
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неверный формат данных")
		_, err := h.bot.Send(msg)
		return err
	}

	selectedDate := parts[2] // 13.10.2025

	response := fmt.Sprintf("🕐 Выберите время отправки для %s:\n\n"+
		"⚠️ Время указывается в часовом поясе UTC+5 (Алматы/Ташкент)", selectedDate)

	// Создаем кнопки с временем как в legacy
	timeButtons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🌅 06:00", fmt.Sprintf("schedule_time_%s_06:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌄 08:00", fmt.Sprintf("schedule_time_%s_08:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("☀️ 10:00", fmt.Sprintf("schedule_time_%s_10:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌞 12:00", fmt.Sprintf("schedule_time_%s_12:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌇 15:00", fmt.Sprintf("schedule_time_%s_15:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌆 18:00", fmt.Sprintf("schedule_time_%s_18:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌃 20:00", fmt.Sprintf("schedule_time_%s_20:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌙 22:00", fmt.Sprintf("schedule_time_%s_22:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🕛 00:00", fmt.Sprintf("schedule_time_%s_00:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏰ Свое время", fmt.Sprintf("schedule_custom_time_%s", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к датам", "schedule_notification"),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(timeButtons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleScheduleTimeCallback обрабатывает выбор времени для планирования
func (h *Handler) HandleScheduleTimeCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Парсим данные: schedule_time_13.10.2025_10:00
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неверный формат данных")
		_, err := h.bot.Send(msg)
		return err
	}

	selectedDate := parts[2] // 13.10.2025
	selectedTime := parts[3] // 10:00

	response := fmt.Sprintf("📢 Выберите тип уведомления для отправки:\n\n"+
		"📅 Дата: %s\n"+
		"🕐 Время: %s (UTC+5)", selectedDate, selectedTime)

	// Создаем кнопки с типами уведомлений как в legacy
	typeButtons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("💌 Мини-дневник", fmt.Sprintf("schedule_type_%s_%s_diary", selectedDate, selectedTime)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("👩🏼‍❤️‍👨🏻 Упражнение недели", fmt.Sprintf("schedule_type_%s_%s_exercise", selectedDate, selectedTime)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("💒 Мотивация", fmt.Sprintf("schedule_type_%s_%s_motivation", selectedDate, selectedTime)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("✏️ Кастомное уведомление", fmt.Sprintf("schedule_type_%s_%s_custom", selectedDate, selectedTime)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к времени", fmt.Sprintf("schedule_date_%s", selectedDate)),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(typeButtons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}
