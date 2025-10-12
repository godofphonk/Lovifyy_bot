package diary

import (
	"fmt"

	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает функциональность дневника
type Handler struct {
	bot         *tgbotapi.BotAPI
	userManager *models.UserManager
}

// NewHandler создает новый обработчик дневника
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager) *Handler {
	return &Handler{
		bot:         bot,
		userManager: userManager,
	}
}

// HandleDiary обрабатывает нажатие кнопки "Мини-дневник" как в legacy
func (h *Handler) HandleDiary(callbackQuery *tgbotapi.CallbackQuery) error {
	response := "📝 Мини дневник\n\n" +
		"Выберите ваш пол для персонализированных советов и подсказок:"

	// Создаем кнопки выбора гендера как в legacy
	buttons := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Парень", "diary_gender_male"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Девушка", "diary_gender_female"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Посмотреть записи", "diary_view"),
		),
	}

	diaryKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = diaryKeyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleDiaryGender обрабатывает выбор пола для дневника
func (h *Handler) HandleDiaryGender(callbackQuery *tgbotapi.CallbackQuery, gender string) error {
	userID := callbackQuery.From.ID
	h.userManager.SetState(userID, "diary")

	var genderEmoji string
	var genderText string
	if gender == "male" {
		genderEmoji = "👨"
		genderText = "парня"
	} else {
		genderEmoji = "👩"
		genderText = "девушки"
	}

	response := fmt.Sprintf("📝 Режим мини дневника активирован для %s %s!\n\n"+
		"Теперь просто напишите свои мысли, заметки или события дня. "+
		"Я буду подтверждать, что ваши записи сохранены.\n\n"+
		"Это ваше личное пространство для записей и размышлений.", genderEmoji, genderText)
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}

// HandleDiaryView обрабатывает просмотр записей дневника
func (h *Handler) HandleDiaryView(callbackQuery *tgbotapi.CallbackQuery) error {
	response := "👀 Просмотр записей дневника\n\n" +
		"Здесь будут отображаться ваши записи из мини-дневника.\n\n" +
		"Функция просмотра записей будет доступна в следующих обновлениях."
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}
