package diary

import (
	"fmt"
	"strings"

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

// HandleDiaryGender обрабатывает выбор пола для дневника - показывает выбор недели
func (h *Handler) HandleDiaryGender(callbackQuery *tgbotapi.CallbackQuery, gender string) error {
	var genderEmoji string
	var genderText string
	if gender == "male" {
		genderEmoji = "👨"
		genderText = "парня"
	} else {
		genderEmoji = "👩"
		genderText = "девушки"
	}

	response := fmt.Sprintf("📝 Мини дневник для %s %s\n\n"+
		"Выберите неделю для записей:", genderEmoji, genderText)

	// Создаем кнопки выбора недели
	buttons := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1️⃣ Неделя 1", fmt.Sprintf("diary_week_%s_1", gender)),
			tgbotapi.NewInlineKeyboardButtonData("2️⃣ Неделя 2", fmt.Sprintf("diary_week_%s_2", gender)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ Неделя 3", fmt.Sprintf("diary_week_%s_3", gender)),
			tgbotapi.NewInlineKeyboardButtonData("4️⃣ Неделя 4", fmt.Sprintf("diary_week_%s_4", gender)),
		),
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleDiaryWeek обрабатывает выбор недели для дневника
func (h *Handler) HandleDiaryWeek(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Парсим данные: diary_week_<gender>_<week>
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return fmt.Errorf("invalid diary week callback data: %s", data)
	}

	gender := parts[2]
	week := parts[3]

	var genderEmoji string
	var genderText string
	if gender == "male" {
		genderEmoji = "👨"
		genderText = "парня"
	} else {
		genderEmoji = "👩"
		genderText = "девушки"
	}

	response := fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n\n"+
		"Выберите тип записи:", genderEmoji, genderText, week)

	// Создаем кнопки типов записей
	buttons := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💭 Личные мысли", fmt.Sprintf("diary_type_%s_%s_personal", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💕 О партнере", fmt.Sprintf("diary_type_%s_%s_partner", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌟 О отношениях", fmt.Sprintf("diary_type_%s_%s_relationship", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Упражнения недели", fmt.Sprintf("diary_type_%s_%s_exercises", gender, week)),
		),
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleDiaryType обрабатывает выбор типа записи в дневнике
func (h *Handler) HandleDiaryType(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Парсим данные: diary_type_<gender>_<week>_<type>
	parts := strings.Split(data, "_")
	if len(parts) < 5 {
		return fmt.Errorf("invalid diary type callback data: %s", data)
	}

	gender := parts[2]
	week := parts[3]
	diaryType := parts[4]

	userID := callbackQuery.From.ID
	// Сохраняем состояние с полной информацией
	h.userManager.SetState(userID, fmt.Sprintf("diary_%s_%s_%s", gender, week, diaryType))

	var genderEmoji string
	var genderText string
	if gender == "male" {
		genderEmoji = "👨"
		genderText = "парня"
	} else {
		genderEmoji = "👩"
		genderText = "девушки"
	}

	var typeText string
	switch diaryType {
	case "personal":
		typeText = "💭 Личные мысли"
	case "partner":
		typeText = "💕 О партнере"
	case "relationship":
		typeText = "🌟 О отношениях"
	case "exercises":
		typeText = "📋 Упражнения недели"
	}

	response := fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
		"Режим записи активирован! Теперь просто напишите свои мысли, заметки или наблюдения. "+
		"Я сохраню вашу запись в соответствующую категорию.\n\n"+
		"Это ваше личное пространство для размышлений.", genderEmoji, genderText, week, typeText)
	
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
