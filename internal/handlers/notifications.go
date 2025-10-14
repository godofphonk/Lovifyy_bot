package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// showNotificationTypeActions показывает действия для выбранного типа
func (ch *CommandHandler) showNotificationTypeActions(userID int64, typ string) error {
	title := map[string]string{
		string(models.NotificationDiary):      "💌 Мини‑дневник",
		string(models.NotificationExercise):   "👩🏼‍❤️‍👨🏻 Упражнения",
		string(models.NotificationMotivation): "💪 Мотивация",
	}[typ]
	if title == "" {
		title = "📢 Уведомление"
	}

	text := fmt.Sprintf("%s\n\nВыберите действие:", title)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Предпросмотр", "notify_preview_"+typ),
			tgbotapi.NewInlineKeyboardButtonData("📤 Отправить всем", "notify_send_all_"+typ),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать", "notify_schedule_"+typ),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "admin_notifications"),
		),
	)
	msg := tgbotapi.NewMessage(userID, text)
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}

// previewNotification — GPT предпросмотр
func (ch *CommandHandler) previewNotification(userID int64, typ string) error {
	nt := models.NotificationType(typ)
	text, err := ch.notificationService.GenerateNotification(nt)
	if err != nil {
		return ch.simpleMsg(userID, fmt.Sprintf("❌ Ошибка генерации: %v", err))
	}
	msg := tgbotapi.NewMessage(userID, "📝 Предпросмотр:\n\n"+text)
	msg.ParseMode = "HTML"
	_, err = ch.bot.Send(msg)
	return err
}

// sendNowNotification — мгновенная отправка
func (ch *CommandHandler) sendNowNotification(userID int64, typ string) error {
	nt := models.NotificationType(typ)
	if err := ch.notificationService.SendInstantNotification(nt, nil); err != nil {
		return ch.simpleMsg(userID, fmt.Sprintf("❌ Ошибка отправки: %v", err))
	}
	return ch.simpleMsg(userID, "✅ Уведомление отправлено всем.")
}

// showSchedulePresets — пресеты времени
func (ch *CommandHandler) showSchedulePresets(userID int64, typ string) error {
	now := time.Now()
	today10 := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location()).Unix()
	today20 := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location()).Unix()
	tomorrow := now.Add(24 * time.Hour)
	tomorrow10 := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, now.Location()).Unix()
	tomorrow20 := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 20, 0, 0, 0, now.Location()).Unix()

	text := "⏰ Выберите время отправки:"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня 10:00", fmt.Sprintf("notify_schedule_preset_%s_%d", typ, today10)),
			tgbotapi.NewInlineKeyboardButtonData("Сегодня 20:00", fmt.Sprintf("notify_schedule_preset_%s_%d", typ, today20)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завтра 10:00", fmt.Sprintf("notify_schedule_preset_%s_%d", typ, tomorrow10)),
			tgbotapi.NewInlineKeyboardButtonData("Завтра 20:00", fmt.Sprintf("notify_schedule_preset_%s_%d", typ, tomorrow20)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "admin_notifications"),
		),
	)
	msg := tgbotapi.NewMessage(userID, text)
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}

// scheduleNotificationAt — запись в расписание
func (ch *CommandHandler) scheduleNotificationAt(userID int64, typ string, at time.Time) error {
	nt := models.NotificationType(typ)
	if _, err := ch.notificationService.ScheduleNotification(at, nt, nil); err != nil {
		return ch.simpleMsg(userID, fmt.Sprintf("❌ Ошибка планирования: %v", err))
	}
	return ch.simpleMsg(userID, fmt.Sprintf("✅ Запланировано на %s", at.Format("02.01 15:04")))
}

// showScheduledNotifications — список задач
func (ch *CommandHandler) showScheduledNotifications(userID int64) error {
	items, err := ch.notificationService.ListScheduled()
	if err != nil {
		return ch.simpleMsg(userID, fmt.Sprintf("❌ Ошибка загрузки: %v", err))
	}
	if len(items) == 0 {
		return ch.simpleMsg(userID, "📋 Запланированных уведомлений нет.")
	}
	
	text := "📋 Запланированные уведомления:\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton
	
	// Часовой пояс UTC+5 для отображения
	utc5 := time.FixedZone("UTC+5", 5*60*60)
	
	for i, it := range items {
		// Конвертируем время в UTC+5 для отображения
		localTime := it.SendAt.In(utc5)
		
		// Определяем тип уведомления
		var typeEmoji, typeName string
		switch string(it.Type) {
		case "diary":
			typeEmoji = "💌"
			typeName = "Мини-дневник"
		case "exercise":
			typeEmoji = "👩🏼‍❤️‍👨🏻"
			typeName = "Упражнение недели"
		case "motivation":
			typeEmoji = "💒"
			typeName = "Мотивация"
		case "custom":
			typeEmoji = "✏️"
			typeName = "Кастомное"
		default:
			typeEmoji = "📢"
			typeName = string(it.Type)
		}
		
		// Заголовок уведомления
		text += fmt.Sprintf("🔹 **Уведомление #%d**\n", i+1)
		text += fmt.Sprintf("📅 **Дата:** %s\n", localTime.Format("02.01.2006 15:04"))
		text += fmt.Sprintf("📢 **Тип:** %s %s\n", typeEmoji, typeName)
		
		// Показываем текст уведомления
		var messageText string
		if it.CustomText != "" {
			// Для кастомных уведомлений показываем кастомный текст
			messageText = it.CustomText
		} else if it.Message != "" {
			// Для стандартных уведомлений показываем сгенерированное сообщение
			messageText = it.Message
		} else {
			messageText = "Текст будет сгенерирован при отправке"
		}
		
		text += fmt.Sprintf("💬 **Текст:** %s\n", messageText)
		text += fmt.Sprintf("🆔 **ID:** `%s`\n\n", it.ID)
		
		// Кнопка отмены
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ Отменить #%d", i+1), "notify_cancel_"+it.ID),
		))
	}
	
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "notifications_menu"),
	))
	
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	_, err = ch.bot.Send(msg)
	return err
}

// cancelScheduledNotification — отмена задачи
func (ch *CommandHandler) cancelScheduledNotification(userID int64, id string) error {
	if err := ch.notificationService.CancelScheduled(id); err != nil {
		return ch.simpleMsg(userID, fmt.Sprintf("❌ Ошибка отмены: %v", err))
	}
	return ch.simpleMsg(userID, "✅ Уведомление отменено.")
}

// simpleMsg — утилита


// handleNotificationCallbacks обрабатывает расширенные колбэки уведомлений
func (ch *CommandHandler) handleNotificationCallbacks(userID int64, data string) error {
	// preview GPT
	if strings.HasPrefix(data, "notify_preview_") {
		typ := strings.TrimPrefix(data, "notify_preview_")
		return ch.previewNotification(userID, typ)
	}
	// send now
	if strings.HasPrefix(data, "notify_send_all_") {
		typ := strings.TrimPrefix(data, "notify_send_all_")
		return ch.sendNowNotification(userID, typ)
	}
	// schedule presets or menu
	if strings.HasPrefix(data, "notify_schedule_") {
		if strings.HasPrefix(data, "notify_schedule_preset_") {
			rest := strings.TrimPrefix(data, "notify_schedule_preset_")
			parts := strings.Split(rest, "_")
			if len(parts) < 2 {
				return ch.simpleMsg(userID, "❌ Неверный пресет расписания")
			}
			typ := parts[0]
			ts, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return ch.simpleMsg(userID, "❌ Неверная дата пресета")
			}
			return ch.scheduleNotificationAt(userID, typ, time.Unix(ts, 0))
		}
		typ := strings.TrimPrefix(data, "notify_schedule_")
		return ch.showSchedulePresets(userID, typ)
	}
	// cancel scheduled
	if strings.HasPrefix(data, "notify_cancel_") {
		id := strings.TrimPrefix(data, "notify_cancel_")
		return ch.cancelScheduledNotification(userID, id)
	}
	return nil
}


