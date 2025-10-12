package exercises

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleWeekAction обрабатывает действия внутри недели как в legacy
func (h *Handler) HandleWeekAction(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Парсим данные: week_<номер>_<действие>
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return fmt.Errorf("invalid week action callback data: %s", data)
	}

	weekStr := parts[1]
	action := parts[2]

	weekNum, err := strconv.Atoi(weekStr)
	if err != nil {
		return fmt.Errorf("invalid week number: %s", weekStr)
	}

	exercise, err := h.exerciseManager.GetWeekExercise(weekNum)
	if err != nil || exercise == nil {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Упражнения для этой недели не найдены")
		_, err := h.bot.Send(msg)
		return err
	}

	var response string

	switch action {
	case "questions":
		if exercise.Questions != "" {
			response = fmt.Sprintf("👩‍❤️‍👨 Упражнения для %d недели\n\n%s", weekNum, exercise.Questions)
		} else {
			response = "👩‍❤️‍👨 Упражнения для этой недели еще не настроены"
		}

	case "tips":
		if exercise.Tips != "" {
			response = fmt.Sprintf("💡 Подсказки для %d недели\n\n%s", weekNum, exercise.Tips)
		} else {
			response = "💡 Подсказки\n\n• Будьте открыты друг с другом\n• Слушайте внимательно\n• Не судите, а поддерживайте\n• Делитесь своими чувствами честно"
		}

	case "insights":
		// Показываем выбор гендера для инсайта как в legacy
		return h.HandleInsightGenderChoice(callbackQuery, weekNum)

	case "joint":
		if exercise.JointQuestions != "" {
			response = fmt.Sprintf("👫 Совместные вопросы для %d недели\n\n%s", weekNum, exercise.JointQuestions)
		} else {
			response = "👫 Совместные вопросы для этой недели еще не настроены"
		}

	case "diary":
		if exercise.DiaryInstructions != "" {
			response = fmt.Sprintf("📝 Что писать в дневнике для %d недели\n\n%s", weekNum, exercise.DiaryInstructions)
		} else {
			response = "📝 Инструкции для дневника этой недели еще не настроены"
		}

	default:
		response = "❓ Неизвестное действие"
	}

	// Добавляем кнопку "Назад к неделе"
	backButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к неделе", fmt.Sprintf("week_%d", weekNum)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = backButton
	_, err = h.bot.Send(msg)
	return err
}
