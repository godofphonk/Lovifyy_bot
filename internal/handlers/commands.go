package handlers

import (
	"fmt"
	"strings"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/handlers/admin"
	"Lovifyy_bot/internal/handlers/chat"
	"Lovifyy_bot/internal/handlers/diary"
	exerciseHandlers "Lovifyy_bot/internal/handlers/exercises"
	"Lovifyy_bot/internal/handlers/scheduling"
	"Lovifyy_bot/internal/history"
	"Lovifyy_bot/internal/models"
	"Lovifyy_bot/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler обрабатывает команды бота (роутер)
type CommandHandler struct {
	bot                 *tgbotapi.BotAPI
	userManager         *models.UserManager
	exerciseManager     *exercises.Manager
	notificationService *services.NotificationService
	historyManager      *history.Manager
	ai                  *ai.OpenAIClient

	// Специализированные обработчики
	adminHandler      *admin.Handler
	exerciseHandler   *exerciseHandlers.Handler
	diaryHandler      *diary.Handler
	chatHandler       *chat.Handler
	schedulingHandler *scheduling.Handler
}

// NewCommandHandler создает новый обработчик команд
func NewCommandHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager, exerciseManager *exercises.Manager, notificationService *services.NotificationService, historyManager *history.Manager, ai *ai.OpenAIClient) *CommandHandler {
	return &CommandHandler{
		bot:                 bot,
		userManager:         userManager,
		exerciseManager:     exerciseManager,
		notificationService: notificationService,
		historyManager:      historyManager,
		ai:                  ai,
		
		// Инициализируем специализированные обработчики
		adminHandler:      admin.NewHandler(bot, userManager, exerciseManager, notificationService),
		exerciseHandler:   exerciseHandlers.NewHandler(bot, userManager, exerciseManager),
		diaryHandler:      diary.NewHandler(bot, userManager),
		chatHandler:       chat.NewHandler(bot, userManager),
		schedulingHandler: scheduling.NewHandler(bot, userManager, notificationService),
	}
}

// HandleStart обрабатывает команду /start точно как в legacy
func (ch *CommandHandler) HandleStart(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	username := update.Message.From.UserName
	
	// Регистрируем пользователя в системе уведомлений
	ch.notificationService.RegisterUser(userID, username)
	ch.notificationService.UpdateUserActivity(userID)
	
	ch.userManager.ClearState(userID)

	// Точное приветственное сообщение из legacy
	welcomeText := "Привет, дорогие! 👋💖 Я так рад видеть вас здесь и вместе отправиться в это маленькое путешествие по вашим отношениям! 🫂\n\n" +
		"Этот чат создан для того, чтобы каждый день находить моменты радости, тепла и взаимопонимания, замечать друг друга и вместе делать ваши отношения ещё более счастливыми. Здесь есть несколько мест, которые помогут вам в этом:\n\n" +
		"1️⃣ Упражнение недели 💑\n" +
		"Каждую неделю я буду предлагать одно задание, которое помогает лучше понимать друг друга, делиться чувствами и развивать приятные привычки общения.\n" +
		"Важно: всё, что вы делаете в упражнениях, нужно фиксировать в мини-дневнике, чтобы видеть свой прогресс и маленькие успехи. 💗\n\n" +
		"2️⃣ Мини-дневник 💌\n" +
		"Это место для ежедневных коротких заметок о ваших наблюдениях, открытиях и шагах в отношениях. Даже одно предложение в день помогает закреплять навыки, видеть рост ваших отношений и отмечать позитивные изменения.\n\n" +
		"💡 Совет: не переживайте о форме или идеальности записей — главное, чтобы это было честно и от сердца. Мини-дневник помогает закреплять всё, чему вы учитесь в упражнениях недели, и видеть положительные изменения в отношениях.\n\n" +
		"3️⃣ Задать вопрос о отношениях 💒\n" +
		"Вы можете написать мне любой вопрос о ваших отношениях в любое время. Я дам совет или подсказку, чтобы общение и взаимопонимание стало ещё теплее. Это работает отдельно от упражнений и дневника, когда захотите. 🫶🏻\n\n" +
		"💌 Совет от меня: наслаждайтесь процессом, замечайте маленькие радости, делитесь впечатлениями и фиксируйте всё в мини-дневнике.\n" +
		"Ваши отношения уникальны, и каждая честная беседа, каждое маленькое внимание друг к другу делает их крепче и теплее. 💒🎀"

	// Создаем простую inline клавиатуру с тремя основными функциями точно как в legacy
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💑 Упражнение недели", "advice"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💌 Мини-дневник", "diary"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💒 Задать вопрос о отношениях", "chat"),
		),
	)

	// Добавляем админские кнопки для администраторов
	if ch.userManager.IsAdmin(userID) {
		adminRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👑 Админ-панель", "adminhelp"),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, adminRow)
	}

	msg := tgbotapi.NewMessage(userID, welcomeText)
	msg.ReplyMarkup = keyboard
	_, err := ch.bot.Send(msg)
	return err
}

// HandleCallback обрабатывает различные callback queries (главный роутер)
func (ch *CommandHandler) HandleCallback(update tgbotapi.Update) error {
	data := update.CallbackQuery.Data

	// Подтверждаем получение callback как в legacy
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	ch.bot.Request(callback)

	// Роутинг по основным категориям
	switch {
	// Основные функции
	case data == "chat":
		return ch.chatHandler.HandleChat(update.CallbackQuery)
	case data == "advice":
		return ch.exerciseHandler.HandleAdvice(update.CallbackQuery)
	case data == "diary":
		return ch.diaryHandler.HandleDiary(update.CallbackQuery)

	// Админ функции
	case data == "adminhelp":
		return ch.adminHandler.HandleAdminHelp(update.CallbackQuery)
	case data == "prompt":
		return ch.adminHandler.HandlePrompt(update.CallbackQuery)
	case data == "setprompt_menu":
		return ch.adminHandler.HandleSetPromptMenu(update.CallbackQuery)
	case data == "welcome":
		return ch.adminHandler.HandleWelcome(update.CallbackQuery)
	case data == "setwelcome_menu":
		return ch.adminHandler.HandleSetWelcomeMenu(update.CallbackQuery)
	case data == "exercises_menu":
		return ch.handleExercisesMenu(update.CallbackQuery)
	case data == "notifications_menu":
		return ch.handleNotificationsMenu(update.CallbackQuery)
	case data == "final_insight_menu":
		return ch.adminHandler.HandleFinalInsightMenu(update.CallbackQuery)
	case data == "generate_final_insight":
		return ch.adminHandler.HandleGenerateFinalInsight(update.CallbackQuery, ch.historyManager, ch.ai)
	case data == "schedule_notification":
		return ch.schedulingHandler.HandleScheduleNotification(update.CallbackQuery)
	case data == "view_notifications":
		return ch.handleViewNotifications(update.CallbackQuery)
	case data == "send_now":
		return ch.handleSendNow(update.CallbackQuery)
	case data == "notify_custom":
		return ch.handleCustomNotification(update.CallbackQuery)
	case data == "notify_schedule_custom":
		return ch.handleScheduleCustomNotification(update.CallbackQuery)

	// Дневник
	case data == "diary_gender_male":
		return ch.diaryHandler.HandleDiaryGender(update.CallbackQuery, "male")
	case data == "diary_gender_female":
		return ch.diaryHandler.HandleDiaryGender(update.CallbackQuery, "female")
	case data == "diary_view":
		return ch.diaryHandler.HandleDiaryView(update.CallbackQuery)

	// Недели упражнений
	case data == "week_1":
		return ch.exerciseHandler.HandleWeek(update.CallbackQuery, 1)
	case data == "week_2":
		return ch.exerciseHandler.HandleWeek(update.CallbackQuery, 2)
	case data == "week_3":
		return ch.exerciseHandler.HandleWeek(update.CallbackQuery, 3)
	case data == "week_4":
		return ch.exerciseHandler.HandleWeek(update.CallbackQuery, 4)

	// Паттерны callback'ов
	case strings.HasPrefix(data, "week_"):
		return ch.exerciseHandler.HandleWeekAction(update.CallbackQuery, data)
	case strings.HasPrefix(data, "insight_"):
		return ch.exerciseHandler.HandleInsightGender(update.CallbackQuery, data)
	case strings.HasPrefix(data, "notify_send_all_"):
		return ch.handleSendAllNotifications(update.CallbackQuery, data)
	case strings.HasPrefix(data, "notify_"):
		return ch.handleNotificationCallbacks(update.CallbackQuery.From.ID, data)
	case strings.HasPrefix(data, "schedule_date_"):
		return ch.schedulingHandler.HandleScheduleDateCallback(update.CallbackQuery, data)
	case strings.HasPrefix(data, "schedule_time_"):
		return ch.schedulingHandler.HandleScheduleTimeCallback(update.CallbackQuery, data)
	case strings.HasPrefix(data, "schedule_type_"):
		return ch.schedulingHandler.HandleScheduleTypeCallback(update.CallbackQuery, data)
	case strings.HasPrefix(data, "schedule_custom_time_"):
		return ch.schedulingHandler.HandleScheduleCustomTimeCallback(update.CallbackQuery, data)
	case data == "schedule_custom_date":
		return ch.schedulingHandler.HandleScheduleCustomDateCallback(update.CallbackQuery)
	case strings.HasPrefix(data, "admin_"):
		return ch.handleLegacyCallbacks(update.CallbackQuery, data)

	default:
		// Обработка legacy callback'ов
		return ch.handleLegacyCallbacks(update.CallbackQuery, data)
	}
}

// handleLegacyCallbacks обрабатывает оставшиеся legacy callback'и
func (ch *CommandHandler) handleLegacyCallbacks(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	// Здесь остаются только сложные callback'и, которые пока не перенесены
	switch {
	case strings.HasPrefix(data, "schedule_date_"):
		return ch.handleScheduleDateCallback(callbackQuery, data)
	case strings.HasPrefix(data, "admin_week_"):
		return ch.handleAdminWeekFieldCallback(callbackQuery, data)
	default:
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❓ Неизвестная команда")
		_, err := ch.bot.Send(msg)
		return err
	}
}

// Временные функции (будут перенесены в соответствующие пакеты)
func (ch *CommandHandler) handleExercisesMenu(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	response := "🗓️ Настройка упражнений\n\nВыберите неделю для настройки упражнений:"
	exercisesKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1️⃣ Неделя", "exercise_week_1"),
			tgbotapi.NewInlineKeyboardButtonData("2️⃣ Неделя", "exercise_week_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ Неделя", "exercise_week_3"),
			tgbotapi.NewInlineKeyboardButtonData("4️⃣ Неделя", "exercise_week_4"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = exercisesKeyboard
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) handleNotificationsMenu(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	text := `📢 <b>Панель уведомлений</b>

Управление системой уведомлений:`

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать", "schedule_notification"),
			tgbotapi.NewInlineKeyboardButtonData("👀 Просмотреть", "view_notifications"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Отправить сейчас", "send_now"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 В админ панель", "adminhelp"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) handleScheduleDateCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	// Парсим дату из callback data: schedule_date_12.10.2025
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неверный формат даты")
		_, err := ch.bot.Send(msg)
		return err
	}

	selectedDate := parts[2] // 12.10.2025

	response := fmt.Sprintf("🕐 Выберите время отправки для %s:\n\n"+
		"⚠️ Время указывается в часовом поясе UTC+5 (Алматы/Ташкент)", selectedDate)

	// Создаем кнопки с временем как в legacy
	timeButtons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🌅 06:00", fmt.Sprintf("schedule_time_%s_06:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌄 08:00", fmt.Sprintf("schedule_time_%s_08:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("☀️ 10:00", fmt.Sprintf("schedule_time_%s_10:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌞 12:00", fmt.Sprintf("schedule_time_%s_12:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌇 15:00", fmt.Sprintf("schedule_time_%s_15:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌆 18:00", fmt.Sprintf("schedule_time_%s_18:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🌃 20:00", fmt.Sprintf("schedule_time_%s_20:00", selectedDate)),
			tgbotapi.NewInlineKeyboardButtonData("🌙 22:00", fmt.Sprintf("schedule_time_%s_22:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🕛 00:00", fmt.Sprintf("schedule_time_%s_00:00", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏰ Свое время", fmt.Sprintf("schedule_custom_time_%s", selectedDate)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к датам", "schedule_notification"),
		},
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(timeButtons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) handleAdminWeekFieldCallback(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "🚧 Функция настройки полей недель временно недоступна")
	_, err := ch.bot.Send(msg)
	return err
}

// handleViewNotifications обрабатывает просмотр уведомлений
func (ch *CommandHandler) handleViewNotifications(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	return ch.showScheduledNotifications(userID)
}

// handleSendNow обрабатывает немедленную отправку уведомлений
func (ch *CommandHandler) handleSendNow(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	text := "📤 Отправить уведомление сейчас\n\nВыберите тип уведомления:"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💌 Мини-дневник", "notify_send_all_diary"),
			tgbotapi.NewInlineKeyboardButtonData("👩🏼‍❤️‍👨🏻 Упражнение недели", "notify_send_all_exercise"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💒 Мотивация", "notify_send_all_motivation"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Кастомное уведомление", "notify_custom"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "notifications_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}

// handleCustomNotification обрабатывает создание кастомного уведомления для немедленной отправки
func (ch *CommandHandler) handleCustomNotification(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	// Устанавливаем состояние для ввода кастомного текста
	ch.userManager.SetState(userID, "custom_notification")

	text := "✏️ Кастомное уведомление\n\n" +
		"Напишите текст уведомления, который будет отправлен всем пользователям.\n\n" +
		"💡 Совет: используйте эмодзи и форматирование для лучшего восприятия."

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "send_now"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}

// handleScheduleCustomNotification обрабатывает создание кастомного уведомления для планирования
func (ch *CommandHandler) handleScheduleCustomNotification(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	// Устанавливаем состояние для ввода кастомного текста для планирования
	ch.userManager.SetState(userID, "custom_notification_schedule")

	text := "✏️ Кастомное уведомление для планирования\n\n" +
		"Напишите текст уведомления, который будет запланирован для отправки.\n\n" +
		"💡 Совет: используйте эмодзи и форматирование для лучшего восприятия."

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "schedule_notification"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = kb
	_, err := ch.bot.Send(msg)
	return err
}


// Остальные методы (HandleHelp, HandleAdmin, etc.) остаются без изменений
func (ch *CommandHandler) HandleHelp(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	text := "ℹ️ Справка по боту"
	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) HandleAdmin(update tgbotapi.Update) error {
	return ch.adminHandler.HandleAdminHelp(&tgbotapi.CallbackQuery{
		From:    update.Message.From,
		Message: update.Message,
	})
}

// simpleMsg отправляет простое сообщение (для совместимости с notifications.go)
func (ch *CommandHandler) simpleMsg(userID int64, text string) error {
	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}

// handleSendAllNotifications обрабатывает отправку уведомлений всем пользователям
func (ch *CommandHandler) handleSendAllNotifications(callbackQuery *tgbotapi.CallbackQuery, data string) error {
	userID := callbackQuery.From.ID

	if !ch.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := ch.bot.Send(msg)
		return err
	}

	var notificationType string
	var typeName string

	switch data {
	case "notify_send_all_diary":
		notificationType = "diary"
		typeName = "💌 Мини-дневник"
	case "notify_send_all_exercise":
		notificationType = "exercise"
		typeName = "👩🏼‍❤️‍👨🏻 Упражнение недели"
	case "notify_send_all_motivation":
		notificationType = "motivation"
		typeName = "💒 Мотивация"
	default:
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Неизвестный тип уведомления")
		_, err := ch.bot.Send(msg)
		return err
	}

	// Отправляем сообщение о начале отправки
	processingMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
		fmt.Sprintf("⏳ Отправляю уведомление %s всем пользователям...", typeName))
	_, err := ch.bot.Send(processingMsg)
	if err != nil {
		return err
	}

	// Генерируем и отправляем уведомление
	var message string
	switch notificationType {
	case "diary":
		message = "📔 <b>Мини-дневник</b>\n\n" +
			"Привет! 😊 Время для записи в дневник!\n\n" +
			"Поделись своими мыслями и чувствами сегодня. " +
			"Что важного произошло? Как твое настроение?\n\n" +
			"Используй команду /diary для записи! ✨"
	case "exercise":
		message = "🏃‍♀️ <b>Упражнения недели</b>\n\n" +
			"Время для полезных упражнений! 💪\n\n" +
			"Попробуй новые активности для улучшения отношений " +
			"и личного развития.\n\n" +
			"Используй команду /advice для получения упражнений! 🌟"
	case "motivation":
		message = "💫 <b>Мотивация дня</b>\n\n" +
			"Каждый день - это новая возможность стать лучше! ✨\n\n" +
			"Помни: маленькие шаги ведут к большим изменениям. " +
			"Ты способен на многое! 🌈\n\n" +
			"Продолжай двигаться вперед! 🚀"
	}

	// Отправляем уведомление всем пользователям
	err = ch.notificationService.SendNotificationToAll(message)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
			fmt.Sprintf("❌ Ошибка при отправке уведомления: %v", err))
		ch.bot.Send(errorMsg)
		return err
	}

	// Получаем количество пользователей
	userCount, _ := ch.notificationService.GetUserCount()

	// Отправляем подтверждение
	confirmText := fmt.Sprintf("✅ Уведомление %s успешно отправлено!\n\n"+
		"👥 Получателей: %d пользователей\n"+
		"📤 Статус: Доставлено", typeName, userCount)
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, confirmText)
	_, err = ch.bot.Send(msg)
	return err
}

// Недостающие методы для совместимости с bot.go
func (ch *CommandHandler) HandleMenu(update tgbotapi.Update) error {
	return ch.HandleStart(update)
}

func (ch *CommandHandler) HandleNotify(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	text := "📢 Система уведомлений временно недоступна"
	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) HandleSetWeek(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	text := "⚙️ Команда /setweek временно недоступна"
	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) HandleAdminHelp(update tgbotapi.Update) error {
	return ch.HandleAdmin(update)
}

func (ch *CommandHandler) HandleUnknownCommand(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	text := "❓ Неизвестная команда. Используйте /help для справки."
	msg := tgbotapi.NewMessage(userID, text)
	_, err := ch.bot.Send(msg)
	return err
}

func (ch *CommandHandler) HandleAdminPanel(update tgbotapi.Update) error {
	return ch.adminHandler.HandleAdminHelp(&tgbotapi.CallbackQuery{
		From:    update.CallbackQuery.From,
		Message: update.CallbackQuery.Message,
	})
}
