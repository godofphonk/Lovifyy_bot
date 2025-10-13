package diary

import (
	"fmt"
	"strings"

	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает функциональность дневника
type Handler struct {
	bot             *tgbotapi.BotAPI
	userManager     *models.UserManager
	exerciseManager *exercises.Manager
}

// NewHandler создает новый обработчик дневника
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, exerciseManager *exercises.Manager) *Handler {
	return &Handler{
		bot:             bot,
		userManager:     userManager,
		exerciseManager: exerciseManager,
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
	// Добавляем логирование для отладки
	fmt.Printf("🔍 HandleDiaryGender called with gender: %s\n", gender)
	
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
	
	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
	h.bot.Send(deleteMsg)
	
	// Отправляем новое сообщение с новыми кнопками
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
			tgbotapi.NewInlineKeyboardButtonData("❓ Ответы на вопросы", fmt.Sprintf("diary_type_%s_%s_questions", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Ответы на совместные вопросы", fmt.Sprintf("diary_type_%s_%s_joint", gender, week)),
		),
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	
	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
	h.bot.Send(deleteMsg)
	
	// Отправляем новое сообщение с новыми кнопками
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleDiaryType обрабатывает выбор типа записи в дневнике
func (h *Handler) HandleDiaryType(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Добавляем логирование для отладки
	fmt.Printf("🔍 HandleDiaryType called with data: %s\n", data)
	
	// Парсим данные: diary_type_<gender>_<week>_<type>
	parts := strings.Split(data, "_")
	if len(parts) < 5 {
		fmt.Printf("❌ Invalid callback data format: %s (parts: %v)\n", data, parts)
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
	var response string
	
	switch diaryType {
	case "personal":
		typeText = "💭 Личные мысли"
		response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
			"Режим записи активирован! Теперь просто напишите свои мысли, заметки или наблюдения. "+
			"Я сохраню вашу запись в соответствующую категорию.\n\n"+
			"Это ваше личное пространство для размышлений.", genderEmoji, genderText, week, typeText)
	
	case "questions":
		typeText = "❓ Ответы на вопросы"
		// Получаем вопросы недели из упражнений
		weekNum := 1
		switch week {
		case "1": weekNum = 1
		case "2": weekNum = 2
		case "3": weekNum = 3
		case "4": weekNum = 4
		}
		
		weekData, err := h.exerciseManager.GetWeekExercise(weekNum)
		if err != nil || weekData == nil {
			response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
				"❌ Не удалось загрузить вопросы для этой недели.\n\n"+
				"Режим записи активирован! Напишите свои ответы на вопросы недели.", 
				genderEmoji, genderText, week, typeText)
		} else {
			response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
				"📋 **Вопросы недели:**\n%s\n\n"+
				"Режим записи активирован! Напишите свои ответы на эти вопросы.", 
				genderEmoji, genderText, week, typeText, weekData.Questions)
		}
	
	case "joint":
		typeText = "👫 Ответы на совместные вопросы"
		// Получаем совместные вопросы недели из упражнений
		weekNum := 1
		switch week {
		case "1": weekNum = 1
		case "2": weekNum = 2
		case "3": weekNum = 3
		case "4": weekNum = 4
		}
		
		weekData, err := h.exerciseManager.GetWeekExercise(weekNum)
		if err != nil || weekData == nil {
			response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
				"❌ Не удалось загрузить совместные вопросы для этой недели.\n\n"+
				"Режим записи активирован! Напишите свои ответы на совместные вопросы недели.", 
				genderEmoji, genderText, week, typeText)
		} else {
			response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
				"👫 **Совместные вопросы недели:**\n%s\n\n"+
				"Режим записи активирован! Напишите свои ответы на эти совместные вопросы.", 
				genderEmoji, genderText, week, typeText, weekData.JointQuestions)
		}
	
	default:
		typeText = "📝 Запись"
		response = fmt.Sprintf("📝 Дневник %s %s - Неделя %s\n%s\n\n"+
			"Режим записи активирован! Теперь просто напишите свои мысли, заметки или наблюдения.", 
			genderEmoji, genderText, week, typeText)
	}
	
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
