package bot

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCustomNotificationMessage обрабатывает ввод текста кастомного уведомления для немедленной отправки
func (b *EnterpriseBot) handleCustomNotificationMessage(userID int64, messageText string) error {
	// Проверяем, что пользователь админ
	if !b.userManager.IsAdmin(userID) {
		b.userManager.ClearState(userID)
		return b.suggestMode(userID)
	}

	// Очищаем состояние
	b.userManager.ClearState(userID)

	// Отправляем кастомное уведомление всем пользователям
	err := b.notificationService.SendCustomNotification(messageText)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, "❌ Ошибка отправки уведомления: "+err.Error())
		b.telegram.Send(msg)
		return err
	}

	// Подтверждаем отправку
	confirmMsg := "✅ Кастомное уведомление отправлено всем пользователям!\n\n📝 Текст:\n" + messageText
	msg := tgbotapi.NewMessage(userID, confirmMsg)
	_, err = b.telegram.Send(msg)
	return err
}

// handleCustomNotificationScheduleMessage обрабатывает ввод текста кастомного уведомления для планирования
func (b *EnterpriseBot) handleCustomNotificationScheduleMessage(userID int64, messageText string) error {
	// Проверяем, что пользователь админ
	if !b.userManager.IsAdmin(userID) {
		b.userManager.ClearState(userID)
		return b.suggestMode(userID)
	}

	// Очищаем состояние
	b.userManager.ClearState(userID)

	// Показываем выбор времени для планирования
	text := "⏰ Выберите время отправки кастомного уведомления:\n\n📝 Текст:\n" + messageText

	// Сохраняем текст во временное хранилище (можно использовать состояние пользователя)
	// Для простоты пока покажем базовые варианты времени
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Через 1 час", "custom_schedule_1h_"+messageText),
			tgbotapi.NewInlineKeyboardButtonData("Через 3 часа", "custom_schedule_3h_"+messageText),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завтра в 10:00", "custom_schedule_tomorrow_"+messageText),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "schedule_notification"),
		),
	)

	msg := tgbotapi.NewMessage(userID, text)
	msg.ReplyMarkup = kb
	_, err := b.telegram.Send(msg)
	return err
}

// handleScheduleCustomTextMessage обрабатывает ввод кастомного текста для планируемого уведомления
func (b *EnterpriseBot) handleScheduleCustomTextMessage(userID int64, messageText string) error {
	// Проверяем, что пользователь админ
	if !b.userManager.IsAdmin(userID) {
		b.userManager.ClearState(userID)
		return b.suggestMode(userID)
	}

	// Очищаем состояние
	b.userManager.ClearState(userID)

	// TODO: Здесь нужно получить сохраненные дату и время из состояния пользователя
	// Пока что показываем подтверждение
	confirmMsg := "✅ Кастомное уведомление для планирования создано!\n\n📝 Текст:\n" + messageText + 
		"\n\n⚠️ Примечание: Для полной реализации нужно сохранять дату/время в состоянии пользователя."
	
	msg := tgbotapi.NewMessage(userID, confirmMsg)
	_, err := b.telegram.Send(msg)
	return err
}

// handleCustomTimeMessage обрабатывает ввод кастомного времени
func (b *EnterpriseBot) handleCustomTimeMessage(userID int64, messageText string, state string) error {
	// Проверяем, что пользователь админ
	if !b.userManager.IsAdmin(userID) {
		b.userManager.ClearState(userID)
		return b.suggestMode(userID)
	}

	// Извлекаем дату из состояния: custom_time_13.10.2025
	parts := strings.Split(state, "_")
	if len(parts) < 3 {
		b.userManager.ClearState(userID)
		msg := tgbotapi.NewMessage(userID, "❌ Ошибка обработки состояния")
		b.telegram.Send(msg)
		return nil
	}

	selectedDate := parts[2] // 13.10.2025

	// Проверяем формат времени
	if !b.isValidTimeFormat(messageText) {
		msg := tgbotapi.NewMessage(userID, "❌ Неверный формат времени. Используйте формат ЧЧ:ММ (например: 14:30)")
		b.telegram.Send(msg)
		return nil
	}

	// Очищаем состояние
	b.userManager.ClearState(userID)

	// Перенаправляем на выбор типа уведомления
	response := fmt.Sprintf("📢 Выберите тип уведомления для отправки:\n\n"+
		"📅 Дата: %s\n"+
		"🕐 Время: %s (UTC+5)", selectedDate, messageText)

	// Создаем кнопки с типами уведомлений
	typeButtons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("💌 Мини-дневник", fmt.Sprintf("schedule_type_%s_%s_diary", selectedDate, messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("👩🏼‍❤️‍👨🏻 Упражнение недели", fmt.Sprintf("schedule_type_%s_%s_exercise", selectedDate, messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("💒 Мотивация", fmt.Sprintf("schedule_type_%s_%s_motivation", selectedDate, messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("✏️ Кастомное уведомление", fmt.Sprintf("schedule_type_%s_%s_custom", selectedDate, messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к времени", fmt.Sprintf("schedule_date_%s", selectedDate)),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(typeButtons...)
	msg := tgbotapi.NewMessage(userID, response)
	msg.ReplyMarkup = keyboard
	_, err := b.telegram.Send(msg)
	return err
}

// handleCustomDateMessage обрабатывает ввод кастомной даты
func (b *EnterpriseBot) handleCustomDateMessage(userID int64, messageText string) error {
	// Проверяем, что пользователь админ
	if !b.userManager.IsAdmin(userID) {
		b.userManager.ClearState(userID)
		return b.suggestMode(userID)
	}

	// Проверяем формат даты
	if !b.isValidDateFormat(messageText) {
		msg := tgbotapi.NewMessage(userID, "❌ Неверный формат даты. Используйте формат ДД.ММ.ГГГГ (например: 15.10.2025)")
		b.telegram.Send(msg)
		return nil
	}

	// Очищаем состояние
	b.userManager.ClearState(userID)

	// Перенаправляем на выбор времени
	response := fmt.Sprintf("🕐 Выберите время отправки для %s:\n\n"+
		"⚠️ Время указывается в часовом поясе UTC+5 (Алматы/Ташкент)", messageText)

	// Создаем кнопки с временем
	timeButtons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🌅 06:00", fmt.Sprintf("schedule_time_%s_06:00", messageText)),
			tgbotapi.NewInlineKeyboardButtonData("🌄 08:00", fmt.Sprintf("schedule_time_%s_08:00", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("☀️ 10:00", fmt.Sprintf("schedule_time_%s_10:00", messageText)),
			tgbotapi.NewInlineKeyboardButtonData("🌞 12:00", fmt.Sprintf("schedule_time_%s_12:00", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌇 15:00", fmt.Sprintf("schedule_time_%s_15:00", messageText)),
			tgbotapi.NewInlineKeyboardButtonData("🌆 18:00", fmt.Sprintf("schedule_time_%s_18:00", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌃 20:00", fmt.Sprintf("schedule_time_%s_20:00", messageText)),
			tgbotapi.NewInlineKeyboardButtonData("🌙 22:00", fmt.Sprintf("schedule_time_%s_22:00", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🕛 00:00", fmt.Sprintf("schedule_time_%s_00:00", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏰ Свое время", fmt.Sprintf("schedule_custom_time_%s", messageText)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к датам", "schedule_notification"),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(timeButtons...)
	msg := tgbotapi.NewMessage(userID, response)
	msg.ReplyMarkup = keyboard
	_, err := b.telegram.Send(msg)
	return err
}

// isValidTimeFormat проверяет формат времени ЧЧ:ММ
func (b *EnterpriseBot) isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

// isValidDateFormat проверяет формат даты ДД.ММ.ГГГГ
func (b *EnterpriseBot) isValidDateFormat(dateStr string) bool {
	_, err := time.Parse("02.01.2006", dateStr)
	return err == nil
}
