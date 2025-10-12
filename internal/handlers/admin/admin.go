package admin

import (
	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/models"
	"Lovifyy_bot/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает админ функциональность
type Handler struct {
	bot                 *tgbotapi.BotAPI
	userManager         *models.UserManager
	exerciseManager     *exercises.Manager
	notificationService *services.NotificationService
}

// NewHandler создает новый обработчик админ функций
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, exerciseManager *exercises.Manager, notificationService *services.NotificationService) *Handler {
	return &Handler{
		bot:                 bot,
		userManager:         userManager,
		exerciseManager:     exerciseManager,
		notificationService: notificationService,
	}
}

// HandleAdminHelp обрабатывает нажатие кнопки "Админ-панель"
func (h *Handler) HandleAdminHelp(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "👑 Админ-панель Lovifyy Bot\n\n" +
		"🔧 Доступные команды:\n" +
		"/setprompt <текст> - изменить системный промпт\n" +
		"/prompt - посмотреть текущий промпт\n" +
		"/setwelcome <текст> - изменить приветственное сообщение\n" +
		"/welcome - посмотреть текущее приветствие\n" +
		"/setweek <неделя> <поле> <значение> - настроить элементы недели\n" +
		"/adminhelp - эта справка\n\n" +
		"💡 Поля для настройки недель:\n" +
		"• title - заголовок недели\n" +
		"• welcome - приветственное сообщение\n" +
		"• questions - вопросы для пары\n" +
		"• tips - подсказки\n" +
		"• insights - инсайты\n" +
		"• joint - совместные вопросы\n" +
		"• diary - инструкции для дневника\n" +
		"• active - открыть/закрыть доступ (true/false)\n\n" +
		"Примеры:\n" +
		"`/setweek 1 title Неделя знакомства`\n" +
		"`/setweek 3 active true` - открыть 3 неделю\n" +
		"`/setweek 2 active false` - закрыть 2 неделю\n\n" +
		"⚠️ Изменения применяются сразу для всех пользователей!"

	// Создаем полную админскую клавиатуру как в legacy
	adminKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Посмотреть промпт", "prompt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить промпт", "setprompt_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👋 Посмотреть приветствие", "welcome"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Изменить приветствие", "setwelcome_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗓️ Настроить упражнения", "exercises_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Уведомления", "notifications_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = adminKeyboard
	_, err := h.bot.Send(msg)
	return err
}

// simpleMsg отправляет простое сообщение
func (h *Handler) simpleMsg(userID int64, text string) error {
	msg := tgbotapi.NewMessage(userID, text)
	_, err := h.bot.Send(msg)
	return err
}
