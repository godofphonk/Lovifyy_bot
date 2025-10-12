package exercises

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleInsightGenderChoice показывает выбор гендера для генерации инсайта как в legacy
func (h *Handler) HandleInsightGenderChoice(callbackQuery *tgbotapi.CallbackQuery, week int) error {
	response := fmt.Sprintf("🔍 Персональный инсайт (%d неделя)\n\n"+
		"Для кого вы хотите получить персональный инсайт?", week)

	// Создаем кнопки выбора гендера
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Для парня", fmt.Sprintf("insight_male_%d", week)),
			tgbotapi.NewInlineKeyboardButtonData("👩 Для девушки", fmt.Sprintf("insight_female_%d", week)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleInsightGender обрабатывает выбор гендера для инсайта
func (h *Handler) HandleInsightGender(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Парсим данные: insight_<gender>_<week>
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return fmt.Errorf("invalid insight callback data: %s", data)
	}

	gender := parts[1]
	weekStr := parts[2]
	weekNum, err := strconv.Atoi(weekStr)
	if err != nil {
		return fmt.Errorf("invalid week number: %s", weekStr)
	}

	var genderName string
	if gender == "male" {
		genderName = "парня"
	} else {
		genderName = "девушки"
	}

	response := fmt.Sprintf("🔍 Персональный инсайт для %s (%d неделя)\n\n"+
		"Для создания персонального инсайта для %s в %d неделе мне нужны записи в дневнике. "+
		"Сначала сделайте записи в дневнике для этой недели, а затем вернитесь к инсайту.\n\n"+
		"📝 Используйте кнопку \"Мини дневник\" для записи мыслей", genderName, weekNum, genderName, weekNum)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err = h.bot.Send(msg)
	return err
}
