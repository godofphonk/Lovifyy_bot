package exercises

import (
	"fmt"

	"github.com/godofphonk/lovifyy-bot/internal/exercises"
	"github.com/godofphonk/lovifyy-bot/internal/models"

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

	// Проверяем активность каждой недели и создаем соответствующие кнопки
	var buttons []tgbotapi.InlineKeyboardButton

	for week := 1; week <= 4; week++ {
		if h.exerciseManager.IsWeekActive(week) {
			// Неделя активна - обычная кнопка
			buttonText := fmt.Sprintf("%d️⃣ Неделя", week)
			callbackData := fmt.Sprintf("week_%d", week)
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData))
		} else {
			// Неделя закрыта - кнопка с замком
			buttonText := fmt.Sprintf("%d️⃣ Неделя 🔒", week)
			callbackData := fmt.Sprintf("week_%d", week) // все равно обрабатывать, чтобы показать сообщение
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData))
		}
	}

	// Создаем клавиатуру с двумя строками
	weekKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons[0], buttons[1]),
		tgbotapi.NewInlineKeyboardRow(buttons[2], buttons[3]),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = weekKeyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleWeek обрабатывает выбор недели упражнений как в legacy
func (h *Handler) HandleWeek(callbackQuery *tgbotapi.CallbackQuery, weekNum int) error {
	// Проверяем, активна ли неделя
	if !h.exerciseManager.IsWeekActive(weekNum) {
		response := fmt.Sprintf("🗓️ %d неделя\n\n🔒 Эта неделя пока недоступна.\n\nАдминистраторы скоро откроют доступ к упражнениям этой недели.", weekNum)
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err := h.bot.Send(msg)
		return err
	}

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
