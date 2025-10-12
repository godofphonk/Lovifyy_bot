package exercises

import (
	"fmt"

	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает функциональность упражнений
type Handler struct {
	bot             *tgbotapi.BotAPI
	userManager     *models.UserManager
	exerciseManager *exercises.Manager
}

// NewHandler создает новый обработчик упражнений
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, exerciseManager *exercises.Manager) *Handler {
	return &Handler{
		bot:             bot,
		userManager:     userManager,
		exerciseManager: exerciseManager,
	}
}

// HandleAdvice обрабатывает нажатие кнопки "Упражнение недели"
func (h *Handler) HandleAdvice(callbackQuery *tgbotapi.CallbackQuery) error {
	response := "🗓️ Выберите неделю для упражнений:\n\n" +
		"Каждая неделя содержит специально подобранные упражнения для укрепления ваших отношений."

	weekKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1️⃣ Неделя", "week_1"),
			tgbotapi.NewInlineKeyboardButtonData("2️⃣ Неделя", "week_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ Неделя", "week_3"),
			tgbotapi.NewInlineKeyboardButtonData("4️⃣ Неделя", "week_4"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = weekKeyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleWeek обрабатывает выбор недели упражнений как в legacy
func (h *Handler) HandleWeek(callbackQuery *tgbotapi.CallbackQuery, weekNum int) error {
	// Получаем упражнения для недели
	exercise, err := h.exerciseManager.GetWeekExercise(weekNum)
	if err != nil {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Извините, произошла ошибка при получении упражнений.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Если упражнения не настроены, показываем сообщение
	if exercise == nil {
		response := fmt.Sprintf("🗓️ Упражнения для %d недели\n\n⚠️ Упражнения для этой недели еще не настроены администраторами.\n\nПожалуйста, обратитесь к администратору или попробуйте позже.", weekNum)
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err := h.bot.Send(msg)
		return err
	}

	// Показываем приветственное сообщение
	welcomeText := exercise.WelcomeMessage
	if welcomeText == "" {
		welcomeText = fmt.Sprintf("Добро пожаловать в %d неделю упражнений!", weekNum)
	}

	response := fmt.Sprintf("%s\n\n%s", exercise.Title, welcomeText)

	// Создаем кнопки для недели как в legacy
	var buttons [][]tgbotapi.InlineKeyboardButton

	if exercise.Questions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💑 Упражнения", fmt.Sprintf("week_%d_questions", weekNum)),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("💡 Подсказки", fmt.Sprintf("week_%d_tips", weekNum)),
	))

	if exercise.Insights != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Инсайт", fmt.Sprintf("week_%d_insights", weekNum)),
		))
	}

	if exercise.JointQuestions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Совместные вопросы", fmt.Sprintf("week_%d_joint", weekNum)),
		))
	}

	if exercise.DiaryInstructions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Что писать в дневнике", fmt.Sprintf("week_%d_diary", weekNum)),
		))
	}

	// Добавляем кнопку "Назад к выбору недель"
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ К выбору недель", "advice"),
	))

	weekKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = weekKeyboard
	_, err = h.bot.Send(msg)
	return err
}
