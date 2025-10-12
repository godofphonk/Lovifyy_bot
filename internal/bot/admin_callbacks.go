package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleExerciseWeekCallback обрабатывает выбор недели для настройки
func (b *EnterpriseBot) handleExerciseWeekCallback(userID int64, week int) error {
	if !b.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ Эта функция доступна только администраторам.")
		_, err := b.telegram.Send(msg)
		return err
	}

	// Получаем текущие упражнения для этой недели
	exercise, err := b.exerciseManager.GetWeekExercise(week)
	if err != nil {
		b.logger.WithError(err).Errorf("Failed to get exercise for week %d", week)
	}

	var status string
	if exercise != nil {
		status = "✅ Настроено"
	} else {
		status = "❌ Не настроено"
	}

	response := fmt.Sprintf("🗓️ Настройка %d недели (%s)\n\n"+
		"Выберите элемент для настройки:", week, status)

	// Создаем кнопки для настройки элементов недели
	adminKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Заголовок", fmt.Sprintf("admin_week_%d_title", week)),
			tgbotapi.NewInlineKeyboardButtonData("👋 Приветствие", fmt.Sprintf("admin_week_%d_welcome", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💪 Упражнения", fmt.Sprintf("admin_week_%d_questions", week)),
			tgbotapi.NewInlineKeyboardButtonData("💡 Подсказки", fmt.Sprintf("admin_week_%d_tips", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Инсайт", fmt.Sprintf("admin_week_%d_insights", week)),
			tgbotapi.NewInlineKeyboardButtonData("👫 Совместные вопросы", fmt.Sprintf("admin_week_%d_joint", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Инструкции для дневника", fmt.Sprintf("admin_week_%d_diary", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Управление доступом", fmt.Sprintf("admin_week_%d_active", week)),
		),
	)

	msg := tgbotapi.NewMessage(userID, response)
	msg.ReplyMarkup = adminKeyboard
	_, err = b.telegram.Send(msg)
	return err
}

// handleAdminWeekFieldCallback обрабатывает настройку полей недели
func (b *EnterpriseBot) handleAdminWeekFieldCallback(userID int64, week int, field string) error {
	if !b.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ Эта функция доступна только администраторам.")
		_, err := b.telegram.Send(msg)
		return err
	}

	var fieldName, example string

	switch field {
	case "title":
		fieldName = "Заголовок"
		example = "/setweek 1 title Неделя знакомства"
	case "welcome":
		fieldName = "Приветственное сообщение"
		example = "/setweek 1 welcome Добро пожаловать в первую неделю!"
	case "questions":
		fieldName = "Упражнения"
		example = "/setweek 1 questions 1. Что вас привлекает в партнере?"
	case "tips":
		fieldName = "Подсказки"
		example = "/setweek 1 tips Будьте честными в ответах"
	case "insights":
		fieldName = "Инсайт"
		example = "/setweek 1 insights Понимание начинается с принятия"
	case "joint":
		fieldName = "Совместные вопросы"
		example = "/setweek 1 joint Обсудите вместе ваши цели"
	case "diary":
		fieldName = "Инструкции для дневника"
		example = "/setweek 1 diary Записывайте свои чувства"
	case "active":
		fieldName = "Активность недели"
		example = "/setweek 1 active true"
	default:
		msg := tgbotapi.NewMessage(userID, "❌ Неизвестное поле")
		_, err := b.telegram.Send(msg)
		return err
	}

	response := fmt.Sprintf("🗓️ Настройка: %s (%d неделя)\n\n"+
		"Используйте команду:\n"+
		"`/setweek %d %s <текст>`\n\n"+
		"Пример:\n"+
		"`%s`", fieldName, week, week, field, example)

	msg := tgbotapi.NewMessage(userID, response)
	_, err := b.telegram.Send(msg)
	return err
}
