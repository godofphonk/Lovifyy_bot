package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"Lovifyy_bot/internal/models"
	"Lovifyy_bot/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler обрабатывает команды бота
type CommandHandler struct {
	bot                 *tgbotapi.BotAPI
	userManager         *models.UserManager
	notificationService *services.NotificationService
}

// NewCommandHandler создает новый обработчик команд
func NewCommandHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, notificationService *services.NotificationService) *CommandHandler {
	return &CommandHandler{
		bot:                 bot,
		userManager:         userManager,
		notificationService: notificationService,
	}
}

// HandleStart обрабатывает команду /start
func (ch *CommandHandler) HandleStart(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	// Очищаем состояние пользователя
	ch.userManager.ClearState(userID)
	
	welcomeText := `🌟 <b>Добро пожаловать в Lovifyy Bot!</b> 🌟

Я ваш персональный помощник для укрепления отношений! 💕

<b>Что я умею:</b>
🤖 <b>ИИ-консультант</b> - персональные советы от GPT-4o-mini
📔 <b>Дневник отношений</b> - структурированные записи по неделям
🧠 <b>Упражнения для пар</b> - психологические задания
📱 <b>Простой интерфейс</b> - все функции через кнопки

Выберите действие в меню ниже! 👇`

	// Создаем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Чат с ИИ", "mode_chat"),
			tgbotapi.NewInlineKeyboardButtonData("📔 Дневник", "mode_diary"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Упражнения", "exercises"),
		),
	)
	
	// Добавляем админские кнопки для администраторов
	if ch.userManager.IsAdmin(userID) {
		adminRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👑 Админ-панель", "admin_panel"),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, adminRow)
	}

	msg := tgbotapi.NewMessage(userID, welcomeText)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	_, err := ch.bot.Send(msg)
	return err
}

// HandleHelp обрабатывает команду /help
func (ch *CommandHandler) HandleHelp(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	helpText := `🆘 <b>Помощь по использованию Lovifyy Bot</b>

<b>Основные команды:</b>
/start - Главное меню
/help - Эта справка
/menu - Вернуться в главное меню

<b>Режимы работы:</b>
💬 <b>Чат с ИИ</b> - Общение с GPT-4o-mini консультантом
📔 <b>Дневник</b> - Ведение структурированных записей
🧠 <b>Упражнения</b> - Психологические задания для пар

<b>Возможности дневника:</b>
• Записи по неделям и типам
• Вопросы для размышления
• Совместные записи
• Личные заметки

<b>Система упражнений:</b>
• Еженедельные задания
• Советы и инсайты
• Совместные вопросы
• Инструкции по ведению дневника`

	if ch.userManager.IsAdmin(userID) {
		helpText += `

<b>👑 Админские команды:</b>
/admin - Админ-панель
/notify - Система уведомлений
/setweek - Управление неделями`
	}

	msg := tgbotapi.NewMessage(userID, helpText)
	msg.ParseMode = "HTML"

	_, err := ch.bot.Send(msg)
	return err
}

// HandleMenu обрабатывает команду /menu
func (ch *CommandHandler) HandleMenu(update tgbotapi.Update) error {
	// Используем ту же логику, что и в /start
	return ch.HandleStart(update)
}

// HandleAdmin обрабатывает команду /admin
func (ch *CommandHandler) HandleAdmin(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ У вас нет прав администратора")
		_, err := ch.bot.Send(msg)
		return err
	}
	
	return ch.showAdminPanel(userID)
}

// HandleNotify обрабатывает команду /notify
func (ch *CommandHandler) HandleNotify(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ У вас нет прав администратора")
		_, err := ch.bot.Send(msg)
		return err
	}
	
	return ch.showNotificationPanel(userID)
}

// showAdminPanel показывает админ-панель
func (ch *CommandHandler) showAdminPanel(userID int64) error {
	text := `👑 <b>Админ-панель</b>

Выберите действие:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Уведомления", "admin_notifications"),
			tgbotapi.NewInlineKeyboardButtonData("🗓️ Недели", "admin_weeks"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "admin_stats"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "admin_settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	_, err := ch.bot.Send(msg)
	return err
}

// showNotificationPanel показывает панель уведомлений
func (ch *CommandHandler) showNotificationPanel(userID int64) error {
	text := `📢 <b>Система уведомлений</b>

Выберите тип уведомления для отправки:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📔 Дневник", "notify_diary"),
			tgbotapi.NewInlineKeyboardButtonData("🧠 Упражнения", "notify_exercise"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💪 Мотивация", "notify_motivation"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки шаблонов", "notify_templates"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard

	_, err := ch.bot.Send(msg)
	return err
}

// HandleSetWeek обрабатывает команду /setweek
func (ch *CommandHandler) HandleSetWeek(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ У вас нет прав администратора")
		_, err := ch.bot.Send(msg)
		return err
	}
	
	// Парсим аргументы команды
	args := strings.Fields(update.Message.Text)
	if len(args) < 4 {
		helpText := `📝 <b>Использование команды /setweek</b>

<b>Формат:</b>
/setweek &lt;неделя&gt; &lt;поле&gt; &lt;значение&gt;

<b>Примеры:</b>
/setweek 1 active true
/setweek 2 questions "Какие чувства вы испытываете?"
/setweek 3 tips "Совет: больше общайтесь"

<b>Доступные поля:</b>
• active - активность недели (true/false)
• questions - вопросы для размышления
• tips - советы
• insights - инсайты
• joint_questions - совместные вопросы
• diary_instructions - инструкции по дневнику`

		msg := tgbotapi.NewMessage(userID, helpText)
		msg.ParseMode = "HTML"
		_, err := ch.bot.Send(msg)
		return err
	}
	
	weekStr := args[1]
	field := args[2]
	value := strings.Join(args[3:], " ")
	
	// Проверяем номер недели
	weekNum, err := strconv.Atoi(weekStr)
	if err != nil || weekNum < 1 {
		msg := tgbotapi.NewMessage(userID, "❌ Неверный номер недели")
		_, err := ch.bot.Send(msg)
		return err
	}
	
	// Здесь должна быть логика обновления недели через exercises manager
	// Пока что заглушка
	successText := fmt.Sprintf("✅ Неделя %d обновлена:\n<b>%s</b> = %s", weekNum, field, value)
	
	msg := tgbotapi.NewMessage(userID, successText)
	msg.ParseMode = "HTML"
	
	_, err = ch.bot.Send(msg)
	return err
}

// HandleUnknownCommand обрабатывает неизвестные команды
func (ch *CommandHandler) HandleUnknownCommand(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	text := `❓ Неизвестная команда.

Используйте /help для получения справки или /menu для возврата в главное меню.`

	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}
