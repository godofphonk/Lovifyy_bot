package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/history"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RateLimiter для ограничения частоты запросов
type RateLimiter struct {
	Users map[int64]time.Time
	Mutex sync.RWMutex
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		Users: make(map[int64]time.Time),
	}
}

// IsAllowed проверяет, можно ли пользователю отправить сообщение
func (rl *RateLimiter) IsAllowed(userID int64) bool {
	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()

	lastMessage, exists := rl.Users[userID]
	now := time.Now()

	// Ограничение: 1 сообщение в 3 секунды
	if exists && now.Sub(lastMessage) < 3*time.Second {
		return false
	}

	rl.Users[userID] = now
	return true
}

// isAdmin проверяет, является ли пользователь администратором
func (b *Bot) isAdmin(userID int64) bool {
	for _, adminID := range b.adminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// setUserState устанавливает состояние пользователя
func (b *Bot) setUserState(userID int64, state string) {
	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()
	b.userStates[userID] = state
}

// getUserState получает состояние пользователя
func (b *Bot) getUserState(userID int64) string {
	b.stateMutex.RLock()
	defer b.stateMutex.RUnlock()
	state, exists := b.userStates[userID]
	if !exists {
		return "" // возвращаем пустое состояние, если не установлено
	}
	return state
}

// Bot представляет Telegram бота с ИИ
type Bot struct {
	telegram        *tgbotapi.BotAPI
	ai              *ai.OpenAIClient
	history         *history.Manager
	exercises       *exercises.Manager
	rateLimiter     *RateLimiter
	systemPrompt    string
	welcomeMessage  string
	adminIDs        []int64
	userStates      map[int64]string // состояния пользователей (chat, diary)
	stateMutex      sync.RWMutex     // мьютекс для безопасного доступа к состояниям
}

// NewBot создает новый экземпляр бота
func NewBot(telegramToken, systemPrompt string, adminIDs []int64) *Bot {
	// Инициализируем Telegram бота
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Ошибка создания Telegram бота:", err)
	}

	bot.Debug = false
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Устанавливаем команды бота (появятся в меню слева)
	log.Println("🔧 Настраиваем команды бота...")

	// Устанавливаем команды для меню
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🚀 Начать работу с ботом"},
		{Command: "chat", Description: "💒 Задать вопрос о отношениях"},
		{Command: "advice", Description: "💑 Упражнение недели"},
		{Command: "diary", Description: "💌 Мини-дневник"},
		{Command: "clear", Description: "🗑️ Очистить историю"},
		{Command: "help", Description: "❓ Справка"},
		{Command: "adminhelp", Description: "👑 Админ-панель"},
	}

	setCommands := tgbotapi.NewSetMyCommands(commands...)
	if _, err := bot.Request(setCommands); err != nil {
		log.Printf("⚠️ Не удалось установить команды: %v", err)
	} else {
		log.Println("✅ Команды для меню установлены!")
	}

	// Инициализируем AI клиента (используем OpenAI)
	aiClient := ai.NewOpenAIClient("gpt-4o-mini")

	// Проверяем доступность AI
	if err := aiClient.TestConnection(); err != nil {
		log.Fatal("AI недоступен:", err)
	}
	log.Println("✅ AI подключен успешно!")

	// Инициализируем систему истории
	historyManager := history.NewManager()
	log.Println("✅ Система истории инициализирована!")

	// Инициализируем менеджер упражнений
	exercisesManager := exercises.NewManager()
	log.Println("✅ Менеджер упражнений инициализирован!")

	// Дефолтное приветственное сообщение
	defaultWelcome := "Привет, дорогие! 👋💖 Я так рад видеть вас здесь и вместе отправиться в это маленькое путешествие по вашим отношениям! 🫂\n\n" +
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

	return &Bot{
		telegram:       bot,
		ai:             aiClient,
		history:        historyManager,
		exercises:      exercisesManager,
		rateLimiter:    NewRateLimiter(),
		systemPrompt:   systemPrompt,
		welcomeMessage: defaultWelcome,
		adminIDs:       adminIDs,
		userStates:     make(map[int64]string),
	}
}

// Start запускает бота
func (b *Bot) Start() {
	// Удаляем webhook перед запуском polling
	del := tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}
	if _, err := b.telegram.Request(del); err != nil {
		log.Printf("Не удалось удалить webhook: %v", err)
	}

	// Ручной polling с offset для избежания дублирования
	offset := 0
	for {
		u := tgbotapi.UpdateConfig{
			Offset:  offset,
			Limit:   100,
			Timeout: 60,
		}

		updates, err := b.telegram.GetUpdates(u)
		if err != nil {
			log.Printf("Ошибка получения апдейтов: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.Message != nil {
				go b.handleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				go b.handleCallbackQuery(update.CallbackQuery)
			}
			offset = update.UpdateID + 1
		}
	}
}

// handleMessage обрабатывает входящие сообщения
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}

	log.Printf("Получено сообщение от %s (ID: %d): %s", username, userID, message.Text)

	// Валидация сообщения
	if !b.validateMessage(message) {
		return
	}

	// Проверка rate limiting
	if !b.rateLimiter.IsAllowed(userID) {
		b.sendMessage(message.Chat.ID, "⏰ Пожалуйста, подождите немного перед отправкой следующего сообщения.")
		return
	}

	// Обработка команд
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Обработка обычных сообщений через ИИ с историей
	b.handleAIMessage(message)
}

// validateMessage проверяет валидность сообщения
func (b *Bot) validateMessage(message *tgbotapi.Message) bool {
	// Проверка на пустое сообщение
	if message.Text == "" {
		return false
	}

	// Ограничение длины сообщения (максимум 4000 символов)
	if len(message.Text) > 4000 {
		b.sendMessage(message.Chat.ID, "❌ Сообщение слишком длинное. Максимум 4000 символов.")
		return false
	}

	// Проверка на спам (повторяющиеся символы)
	if b.isSpamMessage(message.Text) {
		b.sendMessage(message.Chat.ID, "❌ Сообщение выглядит как спам.")
		return false
	}

	return true
}

// isSpamMessage проверяет, является ли сообщение спамом
func (b *Bot) isSpamMessage(text string) bool {
	// Простая проверка на повторяющиеся символы
	if len(text) > 10 {
		charCount := make(map[rune]int)
		for _, char := range text {
			charCount[char]++
		}

		// Если один символ составляет больше 70% сообщения
		for _, count := range charCount {
			if float64(count)/float64(len(text)) > 0.7 {
				return true
			}
		}
	}

	return false
}

// handleCallbackQuery обрабатывает нажатия inline кнопок
func (b *Bot) handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	userID := callbackQuery.From.ID

	log.Printf("Получен callback от пользователя %d: %s", userID, data)

	// Подтверждаем получение callback
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	b.telegram.Request(callback)

	// Обрабатываем callback напрямую по данным
	switch data {
	case "chat":
		b.handleChatCallback(callbackQuery)
	case "advice":
		b.handleAdviceCallback(callbackQuery)
	case "diary":
		b.handleDiaryCallback(callbackQuery)
	case "diary_view":
		b.handleDiaryViewCallback(callbackQuery)
	case "diary_gender_male":
		b.handleDiaryGenderCallback(callbackQuery, "male")
	case "diary_gender_female":
		b.handleDiaryGenderCallback(callbackQuery, "female")
	case "week_1":
		b.handleWeekCallback(callbackQuery, 1)
	case "week_2":
		b.handleWeekCallback(callbackQuery, 2)
	case "week_3":
		b.handleWeekCallback(callbackQuery, 3)
	case "week_4":
		b.handleWeekCallback(callbackQuery, 4)
	case "adminhelp":
		b.handleAdminHelpCallback(callbackQuery)
	case "prompt":
		b.handlePromptCallback(callbackQuery)
	case "setprompt_menu":
		b.handleSetPromptMenuCallback(callbackQuery)
	case "welcome":
		b.handleWelcomeCallback(callbackQuery)
	case "setwelcome_menu":
		b.handleSetWelcomeMenuCallback(callbackQuery)
	case "exercises_menu":
		b.handleExercisesMenuCallback(callbackQuery)
	case "notifications_menu":
		b.handleNotificationsMenuCallback(callbackQuery)
	case "schedule_notification":
		b.handleScheduleNotificationCallback(callbackQuery)
	case "view_notifications":
		b.handleViewNotificationsCallback(callbackQuery)
	case "send_now":
		b.handleSendNowCallback(callbackQuery)
	case "exercise_week_1":
		b.handleExerciseWeekCallback(callbackQuery, 1)
	case "exercise_week_2":
		b.handleExerciseWeekCallback(callbackQuery, 2)
	case "exercise_week_3":
		b.handleExerciseWeekCallback(callbackQuery, 3)
	case "exercise_week_4":
		b.handleExerciseWeekCallback(callbackQuery, 4)
	default:
		// Проверяем, не является ли это callback для выбора даты уведомления
		if strings.HasPrefix(data, "schedule_date_") {
			dateStr := strings.TrimPrefix(data, "schedule_date_")
			b.handleScheduleDateCallback(callbackQuery, dateStr)
			return
		}

		// Проверяем, не является ли это callback для выбора времени уведомления
		if strings.HasPrefix(data, "schedule_time_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 4 {
				dateStr := parts[2]
				timeStr := parts[3]
				b.handleScheduleTimeCallback(callbackQuery, dateStr, timeStr)
				return
			}
		}

		// Проверяем, не является ли это callback для выбора шаблона уведомления
		if strings.HasPrefix(data, "schedule_template_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 5 {
				dateStr := parts[2]
				timeStr := parts[3]
				templateIndex, err := strconv.Atoi(parts[4])
				if err == nil {
					b.handleScheduleTemplateCallback(callbackQuery, dateStr, timeStr, templateIndex)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для мгновенной отправки шаблона
		if strings.HasPrefix(data, "send_now_template_") {
			templateIndex, err := strconv.Atoi(strings.TrimPrefix(data, "send_now_template_"))
			if err == nil {
				b.handleSendNowTemplateCallback(callbackQuery, templateIndex)
				return
			}
		}

		// Проверяем callback для кастомной даты
		if data == "schedule_custom_date" {
			b.handleScheduleCustomDateCallback(callbackQuery)
			return
		}

		// Проверяем callback для кастомного времени
		if strings.HasPrefix(data, "schedule_custom_time_") {
			dateStr := strings.TrimPrefix(data, "schedule_custom_time_")
			b.handleScheduleCustomTimeCallback(callbackQuery, dateStr)
			return
		}

		// Проверяем callback для удаления уведомления
		if strings.HasPrefix(data, "delete_notification_") {
			notificationID, err := strconv.Atoi(strings.TrimPrefix(data, "delete_notification_"))
			if err == nil {
				b.handleDeleteNotificationCallback(callbackQuery, notificationID)
				return
			}
		}

		// Проверяем, не является ли это callback для элементов недели
		if strings.HasPrefix(data, "week_") && strings.Contains(data, "_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 3 {
				week, err := strconv.Atoi(parts[1])
				if err == nil && week >= 1 && week <= 4 {
					action := strings.Join(parts[2:], "_")
					b.handleWeekActionCallback(callbackQuery, week, action)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для админских настроек недели
		if strings.HasPrefix(data, "admin_week_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 4 {
				week, err := strconv.Atoi(parts[2])
				if err == nil && week >= 1 && week <= 4 {
					field := strings.Join(parts[3:], "_")
					b.handleAdminWeekFieldCallback(callbackQuery, week, field)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для дневника с гендером
		if strings.HasPrefix(data, "diary_week_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 4 { // diary_week_[gender]_[week]
				gender := parts[2]
				week, err := strconv.Atoi(parts[3])
				if err == nil && week >= 1 && week <= 4 && (gender == "male" || gender == "female") {
					b.handleDiaryWeekGenderCallback(callbackQuery, gender, week)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для типа записи дневника с гендером
		if strings.HasPrefix(data, "diary_") && strings.Contains(data, "_type_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 5 { // diary_[gender]_[week]_type_[entryType]
				gender := parts[1]
				week, err := strconv.Atoi(parts[2])
				if err == nil && week >= 1 && week <= 4 && (gender == "male" || gender == "female") {
					entryType := strings.Join(parts[4:], "_")
					b.handleDiaryTypeGenderCallback(callbackQuery, gender, week, entryType)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для просмотра записей с гендером
		if strings.HasPrefix(data, "diary_view_gender_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 4 {
				gender := parts[3]
				if gender == "male" || gender == "female" {
					b.handleDiaryViewGenderCallback(callbackQuery, gender)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для просмотра записей недели с гендером
		if strings.HasPrefix(data, "diary_view_week_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 5 { // diary_view_week_[gender]_[week]
				gender := parts[3]
				week, err := strconv.Atoi(parts[4])
				if err == nil && week >= 1 && week <= 4 && (gender == "male" || gender == "female") {
					b.handleDiaryViewWeekGenderCallback(callbackQuery, gender, week)
					return
				}
			}
		}

		// Проверяем, не является ли это callback для инсайта с гендером
		if strings.HasPrefix(data, "insight_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 3 { // insight_[gender]_[week]
				gender := parts[1]
				week, err := strconv.Atoi(parts[2])
				if err == nil && week >= 1 && week <= 4 && (gender == "male" || gender == "female") {
					b.generatePersonalInsightWithGender(callbackQuery, gender, week)
					return
				}
			}
		}
		// Если callback не найден, создаем фейковое сообщение
		fakeMessage := &tgbotapi.Message{
			MessageID: callbackQuery.Message.MessageID,
			From:      callbackQuery.From,
			Chat:      callbackQuery.Message.Chat,
			Date:      callbackQuery.Message.Date,
			Text:      "/" + data,
		}
		b.handleCommand(fakeMessage)
	}
}

// handleChatCallback обрабатывает нажатие кнопки "Обычная беседа"
func (b *Bot) handleChatCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	b.setUserState(userID, "chat")

	response := "💬 Режим обычной беседы активирован!\n\n" +
		"Теперь просто напишите мне любое сообщение, и я отвечу как обычный собеседник. " +
		"Я буду помнить нашу беседу и отвечать в контексте нашего разговора.\n\n" +
		"Чтобы получить упражнения на неделю, используйте /advice"
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleAdviceCallback обрабатывает нажатие кнопки "Упражнения недели"
func (b *Bot) handleAdviceCallback(callbackQuery *tgbotapi.CallbackQuery) {
	// Получаем список активных недель
	activeWeeks := b.exercises.GetActiveWeeks()

	if len(activeWeeks) == 0 {
		response := "🗓️ Упражнения недели\n\n" +
			"⚠️ В данный момент нет доступных недель.\n" +
			"Администраторы еще не открыли доступ к упражнениям."
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	response := "🗓️ Выберите доступную неделю:\n\n" +
		"Каждая неделя содержит специально подобранные упражнения для укрепления ваших отношений."

	// Создаем кнопки только для активных недель
	var buttons [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	weekEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣"}

	for _, week := range activeWeeks {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s Неделя", weekEmojis[week-1]),
			fmt.Sprintf("week_%d", week),
		)
		currentRow = append(currentRow, button)

		// Добавляем по 2 кнопки в ряд
		if len(currentRow) == 2 {
			buttons = append(buttons, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Добавляем оставшиеся кнопки
	if len(currentRow) > 0 {
		buttons = append(buttons, currentRow)
	}

	weekKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = weekKeyboard
	b.telegram.Send(msg)
}

// handleWeekCallback обрабатывает выбор конкретной недели
func (b *Bot) handleWeekCallback(callbackQuery *tgbotapi.CallbackQuery, week int) {
	// Проверяем, активна ли неделя
	if !b.exercises.IsWeekActive(week) {
		response := fmt.Sprintf("🗓️ Упражнения для %d недели\n\n⚠️ Доступ к этой неделе закрыт администраторами.\n\nПожалуйста, выберите доступную неделю.", week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Получаем упражнения для недели
	exercise, err := b.exercises.GetWeekExercise(week)
	if err != nil {
		log.Printf("Ошибка получения упражнений для недели %d: %v", week, err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "Извините, произошла ошибка при получении упражнений.")
		return
	}

	// Если упражнения не настроены, показываем сообщение
	if exercise == nil {
		response := fmt.Sprintf("🗓️ Упражнения для %d недели\n\n⚠️ Упражнения для этой недели еще не настроены администраторами.\n\nПожалуйста, обратитесь к администратору или попробуйте позже.", week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Показываем приветственное сообщение
	welcomeText := exercise.WelcomeMessage
	if welcomeText == "" {
		welcomeText = fmt.Sprintf("Добро пожаловать в %d неделю упражнений!", week)
	}

	response := fmt.Sprintf("%s\n\n%s", exercise.Title, welcomeText)

	// Создаем кнопки для недели
	var buttons [][]tgbotapi.InlineKeyboardButton

	if exercise.Questions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👩‍❤️‍👨 Упражнения", fmt.Sprintf("week_%d_questions", week)),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("💡 Подсказки", fmt.Sprintf("week_%d_tips", week)),
	))

	if exercise.Insights != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Инсайт", fmt.Sprintf("week_%d_insights", week)),
		))
	}

	if exercise.JointQuestions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Совместные вопросы", fmt.Sprintf("week_%d_joint", week)),
		))
	}

	if exercise.DiaryInstructions != "" {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Что писать в дневнике", fmt.Sprintf("week_%d_diary", week)),
		))
	}

	weekKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = weekKeyboard
	b.telegram.Send(msg)
}

// handleWeekActionCallback обрабатывает действия внутри недели
func (b *Bot) handleWeekActionCallback(callbackQuery *tgbotapi.CallbackQuery, week int, action string) {
	exercise, err := b.exercises.GetWeekExercise(week)
	if err != nil || exercise == nil {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Упражнения для этой недели не найдены")
		return
	}

	var response string

	switch action {
	case "questions":
		if exercise.Questions != "" {
			response = fmt.Sprintf("💪 Упражнения для %d недели\n\n%s", week, exercise.Questions)
		} else {
			response = "💪 Упражнения для этой недели еще не настроены"
		}

	case "tips":
		if exercise.Tips != "" {
			response = fmt.Sprintf("💡 Подсказки для %d недели\n\n%s", week, exercise.Tips)
		} else {
			response = "💡 Подсказки\n\n• Будьте открыты друг с другом\n• Слушайте внимательно\n• Не судите, а поддерживайте\n• Делитесь своими чувствами честно"
		}

	case "insights":
		// Показываем выбор гендера для инсайта
		b.handleInsightGenderChoice(callbackQuery, week)
		return

	case "joint":
		if exercise.JointQuestions != "" {
			response = fmt.Sprintf("👫 Совместные вопросы для %d недели\n\n%s", week, exercise.JointQuestions)
		} else {
			response = "👫 Совместные вопросы для этой недели еще не настроены"
		}

	case "diary":
		if exercise.DiaryInstructions != "" {
			response = fmt.Sprintf("📝 Что писать в дневнике (%d неделя)\n\n%s", week, exercise.DiaryInstructions)
		} else {
			response = "📝 Инструкции для дневника еще не настроены"
		}

	default:
		response = "❌ Неизвестное действие"
	}

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleDiaryCallback обрабатывает нажатие кнопки "Мини дневник"
func (b *Bot) handleDiaryCallback(callbackQuery *tgbotapi.CallbackQuery) {
	// Получаем список активных недель
	activeWeeks := b.exercises.GetActiveWeeks()

	if len(activeWeeks) == 0 {
		response := "📝 Мини дневник\n\n" +
			"⚠️ В данный момент нет доступных недель для записей.\n" +
			"Администраторы еще не открыли доступ к неделям."
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	response := "📝 Мини дневник\n\n" +
		"Сначала выберите, за кого вы хотите заполнить дневник:"

	// Создаем кнопки выбора гендера
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
	b.telegram.Send(msg)
}

// handleDiaryWeekCallback обрабатывает выбор недели для дневника
func (b *Bot) handleDiaryWeekCallback(callbackQuery *tgbotapi.CallbackQuery, week int) {
	// Проверяем, активна ли неделя
	if !b.exercises.IsWeekActive(week) {
		response := fmt.Sprintf("📝 Дневник - %d неделя\n\n⚠️ Доступ к записям этой недели закрыт администраторами.\n\nПожалуйста, выберите доступную неделю.", week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	response := fmt.Sprintf("📝 Дневник - %d неделя\n\n"+
		"Выберите тип записи:", week)

	// Создаем кнопки для типов записей
	typeKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Ответы на вопросы", fmt.Sprintf("diary_%d_type_questions", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Ответы на совместные вопросы", fmt.Sprintf("diary_%d_type_joint", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💭 Личные записи и мысли", fmt.Sprintf("diary_%d_type_personal", week)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = typeKeyboard
	b.telegram.Send(msg)
}

// handleDiaryTypeCallback обрабатывает выбор типа записи дневника
func (b *Bot) handleDiaryTypeCallback(callbackQuery *tgbotapi.CallbackQuery, week int, entryType string) {
	userID := callbackQuery.From.ID

	// Устанавливаем состояние пользователя для дневника
	b.setUserState(userID, fmt.Sprintf("diary_%d_%s", week, entryType))

	var response string
	var typeName string

	switch entryType {
	case "questions":
		typeName = "Ответы на вопросы"
		// Получаем вопросы для этой недели
		exercise, err := b.exercises.GetWeekExercise(week)
		if err == nil && exercise != nil && exercise.Questions != "" {
			response = fmt.Sprintf("❓ %s (%d неделя)\n\n"+
				"Напоминание вопросов:\n%s\n\n"+
				"Теперь напишите свои ответы на эти вопросы:", typeName, week, exercise.Questions)
		} else {
			response = fmt.Sprintf("❓ %s (%d неделя)\n\n"+
				"Напишите свои ответы на вопросы недели:", typeName, week)
		}

	case "joint":
		typeName = "Ответы на совместные вопросы"
		// Получаем совместные вопросы для этой недели
		exercise, err := b.exercises.GetWeekExercise(week)
		if err == nil && exercise != nil && exercise.JointQuestions != "" {
			response = fmt.Sprintf("👫 %s (%d неделя)\n\n"+
				"Напоминание совместных вопросов:\n%s\n\n"+
				"Теперь напишите ваши совместные ответы и обсуждения:", typeName, week, exercise.JointQuestions)
		} else {
			response = fmt.Sprintf("👫 %s (%d неделя)\n\n"+
				"Напишите ваши ответы на совместные вопросы:", typeName, week)
		}

	case "personal":
		typeName = "Личные записи и мысли"
		// Получаем инструкции для дневника
		exercise, err := b.exercises.GetWeekExercise(week)
		if err == nil && exercise != nil && exercise.DiaryInstructions != "" {
			response = fmt.Sprintf("💭 %s (%d неделя)\n\n"+
				"Рекомендации для записей:\n%s\n\n"+
				"Напишите свои личные мысли и размышления:", typeName, week, exercise.DiaryInstructions)
		} else {
			response = fmt.Sprintf("💭 %s (%d неделя)\n\n"+
				"Напишите свои личные мысли и размышления:", typeName, week)
		}

	default:
		response = "❌ Неизвестный тип записи"
	}

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleAdminHelpCallback обрабатывает нажатие кнопки "Админ-панель"
func (b *Bot) handleAdminHelpCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}

	response := "👑 Админ-панель Lovifyy Bot\n\n" +
		"🔧 Доступные команды:\n" +
		"/setprompt <текст> - изменить системный промпт\n" +
		"/prompt - посмотреть текущий промпт\n" +
		"/adminhelp - эта справка\n\n" +
		"💡 Примеры промптов:\n" +
		"• Ты дружелюбный помощник\n" +
		"• Ты опытный психолог\n" +
		"• Ты программист-эксперт\n\n" +
		"⚠️ Изменения применяются сразу для всех пользователей!"

	// Создаем админскую клавиатуру
	adminKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Посмотреть промпт", "prompt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить промпт", "setprompt_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = adminKeyboard
	b.telegram.Send(msg)
}

// handlePromptCallback обрабатывает нажатие кнопки "Посмотреть промпт"
func (b *Bot) handlePromptCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}

	response := fmt.Sprintf("🤖 Текущий системный промпт:\n\n%s\n\n💡 Для изменения используйте:\n/setprompt <новый промпт>", b.systemPrompt)
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleSetPromptMenuCallback обрабатывает нажатие кнопки "Изменить промпт"
func (b *Bot) handleSetPromptMenuCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}

	response := "✏️ Изменение системного промпта\n\n" +
		"Отправьте команду в формате:\n" +
		"`/setprompt <новый промпт>`\n\n" +
		"💡 Готовые варианты:\n\n" +
		"Психолог:\n" +
		"`/setprompt Ты опытный психолог, который помогает людям с их личными проблемами. Будь сочувствующим и давай полезные советы.`\n\n" +
		"Дружелюбный помощник:\n" +
		"`/setprompt Ты дружелюбный помощник, готовый ответить на любые вопросы. Будь позитивным и полезным.`\n\n" +
		"Программист:\n" +
		"`/setprompt Ты программист-эксперт, специализирующийся на Go и веб-разработке. Помогай с кодом и объясняй концепции.`"
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleWelcomeCallback обрабатывает нажатие кнопки "Посмотреть приветствие"
func (b *Bot) handleWelcomeCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}

	response := fmt.Sprintf("👋 Текущее приветственное сообщение:\n\n%s\n\n💡 Для изменения используйте:\n/setwelcome <новое приветствие>", b.welcomeMessage)
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleSetWelcomeMenuCallback обрабатывает нажатие кнопки "Изменить приветствие"
func (b *Bot) handleSetWelcomeMenuCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}

	response := "📝 Изменение приветственного сообщения\n\n" +
		"Отправьте команду в формате:\n" +
		"`/setwelcome <новое приветствие>`\n\n" +
		"💡 Готовые варианты:\n\n" +
		"Стандартное:\n" +
		"`/setwelcome Привет! 👋 Я Lovifyy Bot - ваш персональный помощник!`\n\n" +
		"Для пар:\n" +
		"`/setwelcome Добро пожаловать в Lovifyy Bot! ❤️ Я помогу укрепить ваши отношения через упражнения и дневник.`\n\n" +
		"Краткое:\n" +
		"`/setwelcome Привет! Выберите режим работы:`"
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleExercisesMenuCallback обрабатывает нажатие кнопки "Настроить упражнения"
func (b *Bot) handleExercisesMenuCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := "🗓️ Настройка упражнений\n\n" +
		"Выберите неделю для настройки упражнений:"

	// Создаем клавиатуру с выбором недель для настройки
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
	b.telegram.Send(msg)
}

// handleNotificationsMenuCallback обрабатывает нажатие кнопки "Уведомления"
func (b *Bot) handleNotificationsMenuCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := "📢 Управление уведомлениями\n\n" +
		"🕐 Часовой пояс: UTC+5 (Алматы/Ташкент)\n" +
		"📅 Формат времени: ДД.ММ.ГГГГ ЧЧ:ММ\n\n" +
		"Выберите действие:"

	// Создаем клавиатуру с опциями уведомлений
	notificationsKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать уведомление", "schedule_notification"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Посмотреть запланированные", "view_notifications"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Отправить сейчас", "send_now"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = notificationsKeyboard
	b.telegram.Send(msg)
}

// handleScheduleNotificationCallback обрабатывает запланированные уведомления
func (b *Bot) handleScheduleNotificationCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := "⏰ Запланировать уведомление\n\n" +
		"🗓️ Выберите дату отправки:\n" +
		"🕐 Часовой пояс: UTC+5 (Алматы/Ташкент)"

	// Создаем кнопки с датами (сегодня + следующие 6 дней) в UTC+5
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Получаем текущее время в UTC+5 (Алматы/Ташкент)
	location, _ := time.LoadLocation("Asia/Almaty")
	nowUTC5 := time.Now().In(location)
	
	for i := 0; i < 7; i++ {
		date := nowUTC5.AddDate(0, 0, i)
		dateStr := date.Format("02.01.2006")
		var dayName string
		
		switch i {
		case 0:
			dayName = "Сегодня"
		case 1:
			dayName = "Завтра"
		default:
			dayName = date.Format("Mon")
		}
		
		buttonText := fmt.Sprintf("%s (%s)", dayName, dateStr)
		callbackData := fmt.Sprintf("schedule_date_%s", dateStr)
		
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	// Добавляем кнопку для ввода своей даты
	customDateButton := tgbotapi.NewInlineKeyboardButtonData("📅 Своя дата", "schedule_custom_date")
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{customDateButton})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	b.telegram.Send(msg)
}

// handleScheduleDateCallback обрабатывает выбор даты для уведомления
func (b *Bot) handleScheduleDateCallback(callbackQuery *tgbotapi.CallbackQuery, dateStr string) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := fmt.Sprintf("🕐 Выберите время отправки\n\n📅 Дата: %s\n🕐 Часовой пояс: UTC+5", dateStr)

	// Создаем кнопки с временем (каждые 2 часа)
	var buttons [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton
	
	times := []string{"08:00", "10:00", "12:00", "14:00", "16:00", "18:00", "20:00", "22:00"}
	
	for i, timeStr := range times {
		callbackData := fmt.Sprintf("schedule_time_%s_%s", dateStr, timeStr)
		button := tgbotapi.NewInlineKeyboardButtonData(timeStr, callbackData)
		currentRow = append(currentRow, button)
		
		// Добавляем по 2 кнопки в ряд
		if len(currentRow) == 2 || i == len(times)-1 {
			buttons = append(buttons, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Добавляем кнопку для ввода своего времени
	customTimeButton := tgbotapi.NewInlineKeyboardButtonData("🕐 Свое время", fmt.Sprintf("schedule_custom_time_%s", dateStr))
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{customTimeButton})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	b.telegram.Send(msg)
}

// handleScheduleCustomDateCallback обрабатывает ввод кастомной даты
func (b *Bot) handleScheduleCustomDateCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := "📅 Введите дату в формате ДД.ММ.ГГГГ\n\n" +
		"Примеры:\n" +
		"• 15.10.2025\n" +
		"• 01.12.2025\n\n" +
		"🕐 Часовой пояс: UTC+5 (Алматы/Ташкент)"

	// Сохраняем состояние для ввода кастомной даты
	b.setUserState(userID, "notification_custom_date")
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleScheduleCustomTimeCallback обрабатывает ввод кастомного времени
func (b *Bot) handleScheduleCustomTimeCallback(callbackQuery *tgbotapi.CallbackQuery, dateStr string) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := fmt.Sprintf("🕐 Введите время в формате ЧЧ:ММ\n\n📅 Дата: %s\n\n" +
		"Примеры:\n" +
		"• 09:30\n" +
		"• 15:45\n" +
		"• 21:00\n\n" +
		"🕐 Часовой пояс: UTC+5 (Алматы/Ташкент)", dateStr)

	// Сохраняем состояние для ввода кастомного времени
	b.setUserState(userID, fmt.Sprintf("notification_custom_time_%s", dateStr))
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleScheduleTimeCallback обрабатывает выбор времени для уведомления
func (b *Bot) handleScheduleTimeCallback(callbackQuery *tgbotapi.CallbackQuery, dateStr, timeStr string) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := fmt.Sprintf("💌 Выберите шаблон сообщения\n\n📅 Дата: %s\n🕐 Время: %s (UTC+5)", dateStr, timeStr)

	// Создаем кнопки с шаблонами сообщений
	templates := []struct {
		text     string
		template string
	}{
		{"❤️ Напоминание о дневнике", "Привет! ❤️ Не забудьте заполнить дневник сегодня. Ваши мысли и чувства важны для укрепления отношений!"},
		{"💑 Упражнения недели", "Время для упражнений! 💑 Новые задания помогут вам лучше понять друг друга."},
		{"🌟 Мотивация", "Каждый день - это новая возможность стать ближе! 🌟 Цените моменты вместе."},
		{"📝 Свой текст", "custom"},
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for i, tmpl := range templates {
		callbackData := fmt.Sprintf("schedule_template_%s_%s_%d", dateStr, timeStr, i)
		button := tgbotapi.NewInlineKeyboardButtonData(tmpl.text, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	b.telegram.Send(msg)
}

// handleScheduleTemplateCallback обрабатывает выбор шаблона для уведомления
func (b *Bot) handleScheduleTemplateCallback(callbackQuery *tgbotapi.CallbackQuery, dateStr, timeStr string, templateIndex int) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	if templateIndex == 3 {
		// Кастомный текст - просим ввести
		response := fmt.Sprintf("📝 Введите свой текст уведомления\n\n📅 Дата: %s\n🕐 Время: %s (UTC+5)\n\n" +
			"Просто напишите сообщение следующим сообщением:", dateStr, timeStr)
		
		// Сохраняем состояние для ввода кастомного текста
		b.setUserState(userID, fmt.Sprintf("notification_custom_%s_%s", dateStr, timeStr))
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Генерируем только нужный шаблон для оптимизации скорости
	var messageText string
	switch templateIndex {
	case 0:
		messageText = b.generateNotificationTemplate("diary")
	case 1:
		messageText = b.generateNotificationTemplate("exercises")
	case 2:
		messageText = b.generateNotificationTemplate("motivation")
	default:
		messageText = "Привет! ❤️ Напоминание от вашего бота!"
	}

	if templateIndex >= 0 && templateIndex <= 2 {
		
		// Сохраняем уведомление в файл
		if err := b.saveNotification(dateStr, timeStr, messageText); err != nil {
			log.Printf("Ошибка сохранения уведомления: %v", err)
			b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка сохранения уведомления")
			return
		}
		
		response := fmt.Sprintf("✅ Уведомление запланировано!\n\n📅 Дата: %s\n🕐 Время: %s (UTC+5)\n\n💌 Текст:\n%s\n\n" +
			"⚠️ Уведомление будет отправлено всем пользователям бота", dateStr, timeStr, messageText)
		
		log.Printf("👑 Администратор %d запланировал уведомление на %s %s: %s", userID, dateStr, timeStr, messageText)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
	}
}

// ScheduledNotification представляет запланированное уведомление
type ScheduledNotification struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Message  string `json:"message"`
	Created  string `json:"created"`
}

// handleViewNotificationsCallback показывает запланированные уведомления
func (b *Bot) handleViewNotificationsCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	log.Printf("👑 Администратор %d запросил просмотр уведомлений", userID)

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	// Читаем запланированные уведомления из файла
	notifications, err := b.loadScheduledNotifications()
	if err != nil {
		log.Printf("Ошибка загрузки уведомлений: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка загрузки уведомлений")
		return
	}
	
	log.Printf("📋 Загружено %d уведомлений", len(notifications))

	if len(notifications) == 0 {
		response := "📋 Запланированные уведомления\n\n" +
			"📭 Нет запланированных уведомлений\n\n" +
			"Используйте кнопку 'Запланировать уведомление' для создания нового."
		
		log.Printf("📤 Отправляем сообщение о пустом списке")
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err := b.telegram.Send(msg)
		if err != nil {
			log.Printf("❌ Ошибка отправки сообщения о пустом списке: %v", err)
		} else {
			log.Printf("✅ Сообщение о пустом списке отправлено успешно")
		}
		return
	}

	response := "📋 Запланированные уведомления\n\n"
	var buttons [][]tgbotapi.InlineKeyboardButton

	for _, notification := range notifications {
		// Показываем полное сообщение без обрезки
		messagePreview := notification.Message
		
		response += fmt.Sprintf("🔔 ID: %d\n📅 %s в %s\n💌 %s\n\n", 
			notification.ID, notification.Date, notification.Time, b.cleanUTF8(messagePreview))
		
		// Добавляем кнопку для удаления каждого уведомления
		deleteButton := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🗑️ Удалить #%d", notification.ID), 
			fmt.Sprintf("delete_notification_%d", notification.ID))
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{deleteButton})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	
	log.Printf("📤 Отправляем ответ с %d кнопками", len(buttons))
	_, err = b.telegram.Send(msg)
	if err != nil {
		log.Printf("❌ Ошибка отправки сообщения: %v", err)
	} else {
		log.Printf("✅ Сообщение отправлено успешно")
	}
}

// loadScheduledNotifications загружает запланированные уведомления из файла
func (b *Bot) loadScheduledNotifications() ([]ScheduledNotification, error) {
	filename := "scheduled_notifications.json"
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduledNotification{}, nil
		}
		return nil, err
	}

	var notifications []ScheduledNotification
	if err := json.Unmarshal(data, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleDeleteNotificationCallback обрабатывает удаление уведомления
func (b *Bot) handleDeleteNotificationCallback(callbackQuery *tgbotapi.CallbackQuery, notificationID int) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	// Загружаем текущие уведомления
	notifications, err := b.loadScheduledNotifications()
	if err != nil {
		log.Printf("Ошибка загрузки уведомлений: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка загрузки уведомлений")
		return
	}

	// Ищем и удаляем уведомление
	var updatedNotifications []ScheduledNotification
	var deletedNotification *ScheduledNotification
	
	for _, notification := range notifications {
		if notification.ID == notificationID {
			deletedNotification = &notification
		} else {
			updatedNotifications = append(updatedNotifications, notification)
		}
	}

	if deletedNotification == nil {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Уведомление не найдено")
		return
	}

	// Сохраняем обновленный список
	if err := b.saveScheduledNotifications(updatedNotifications); err != nil {
		log.Printf("Ошибка сохранения уведомлений: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка сохранения")
		return
	}

	response := fmt.Sprintf("✅ Уведомление удалено!\n\n🔔 ID: %d\n📅 %s в %s\n💌 %s", 
		deletedNotification.ID, deletedNotification.Date, deletedNotification.Time, deletedNotification.Message)
	
	log.Printf("👑 Администратор %d удалил уведомление ID %d", userID, notificationID)
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// saveNotification сохраняет новое уведомление
func (b *Bot) saveNotification(dateStr, timeStr, messageText string) error {
	// Загружаем существующие уведомления
	notifications, err := b.loadScheduledNotifications()
	if err != nil {
		return err
	}

	// Генерируем новый ID
	maxID := 0
	for _, notification := range notifications {
		if notification.ID > maxID {
			maxID = notification.ID
		}
	}

	// Создаем новое уведомление с UTC+5 временем создания
	location := time.FixedZone("UTC+5", 5*60*60)
	now := time.Now().In(location)
	
	newNotification := ScheduledNotification{
		ID:      maxID + 1,
		Date:    dateStr,
		Time:    timeStr,
		Message: messageText,
		Created: now.Format("02.01.2006 15:04"),
	}

	// Добавляем к списку
	notifications = append(notifications, newNotification)

	// Сохраняем
	return b.saveScheduledNotifications(notifications)
}

// saveScheduledNotifications сохраняет уведомления в файл
func (b *Bot) saveScheduledNotifications(notifications []ScheduledNotification) error {
	filename := "scheduled_notifications.json"
	
	data, err := json.MarshalIndent(notifications, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// handleSendNowCallback обрабатывает отправку уведомления сейчас
func (b *Bot) handleSendNowCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	response := "📤 Отправить уведомление сейчас\n\n💌 Выберите шаблон сообщения:"

	// Создаем кнопки с шаблонами для мгновенной отправки
	templates := []struct {
		text     string
		template string
	}{
		{"❤️ Напоминание о дневнике", "Привет! ❤️ Не забудьте заполнить дневник сегодня. Ваши мысли и чувства важны для укрепления отношений!"},
		{"💑 Упражнения недели", "Время для упражнений! 💑 Новые задания помогут вам лучше понять друг друга."},
		{"🌟 Мотивация", "Каждый день - это новая возможность стать ближе! 🌟 Цените моменты вместе."},
		{"📝 Свой текст", "custom"},
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for i, tmpl := range templates {
		callbackData := fmt.Sprintf("send_now_template_%d", i)
		button := tgbotapi.NewInlineKeyboardButtonData(tmpl.text, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	b.telegram.Send(msg)
}

// handleSendNowTemplateCallback обрабатывает мгновенную отправку шаблона
func (b *Bot) handleSendNowTemplateCallback(callbackQuery *tgbotapi.CallbackQuery, templateIndex int) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	if templateIndex == 3 {
		// Кастомный текст - просим ввести
		response := "📝 Введите текст для мгновенной отправки\n\n" +
			"Просто напишите сообщение следующим сообщением:"
		
		// Сохраняем состояние для ввода кастомного текста
		b.setUserState(userID, "broadcast_custom")
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Генерируем только нужный шаблон для оптимизации скорости
	var messageText string
	switch templateIndex {
	case 0:
		messageText = b.generateNotificationTemplate("diary")
	case 1:
		messageText = b.generateNotificationTemplate("exercises")
	case 2:
		messageText = b.generateNotificationTemplate("motivation")
	default:
		messageText = "Привет! ❤️ Напоминание от вашего бота!"
	}

	if templateIndex >= 0 && templateIndex <= 2 {
		
		// Отправляем уведомление всем пользователям
		sentCount, err := b.broadcastMessage(messageText)
		if err != nil {
			log.Printf("Ошибка отправки уведомления: %v", err)
			b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка отправки уведомления")
			return
		}
		
		response := fmt.Sprintf("✅ Уведомление отправлено!\n\n💌 Текст:\n%s\n\n" +
			"📤 Сообщение отправлено %d пользователям", messageText, sentCount)
		
		log.Printf("👑 Администратор %d отправил мгновенное уведомление %d пользователям", userID, sentCount)
		log.Printf("📝 Полный текст мгновенного уведомления: %s", messageText)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
	}
}

// handleExerciseWeekCallback обрабатывает выбор недели для настройки
func (b *Bot) handleExerciseWeekCallback(callbackQuery *tgbotapi.CallbackQuery, week int) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	// Получаем текущие упражнения для этой недели
	exercise, err := b.exercises.GetWeekExercise(week)
	if err != nil {
		log.Printf("Ошибка получения упражнений для недели %d: %v", week, err)
	}

	var status string
	if exercise != nil {
		status = "✅ Настроено"
	} else {
		status = "❌ Не настроено"
	}

	response := fmt.Sprintf("🗓️ Настройка %d недели (%s)\n\n"+
		"Выберите элемент для настройки:", week, status)

	// Создаем кнопки для настройки элементов недели
	adminKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Заголовок", fmt.Sprintf("admin_week_%d_title", week)),
			tgbotapi.NewInlineKeyboardButtonData("👋 Приветствие", fmt.Sprintf("admin_week_%d_welcome", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💪 Упражнения", fmt.Sprintf("admin_week_%d_questions", week)),
			tgbotapi.NewInlineKeyboardButtonData("💡 Подсказки", fmt.Sprintf("admin_week_%d_tips", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Инсайт", fmt.Sprintf("admin_week_%d_insights", week)),
			tgbotapi.NewInlineKeyboardButtonData("👫 Совместные вопросы", fmt.Sprintf("admin_week_%d_joint", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Инструкции для дневника", fmt.Sprintf("admin_week_%d_diary", week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Управление доступом", fmt.Sprintf("admin_week_%d_active", week)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = adminKeyboard
	b.telegram.Send(msg)
}

// handleAdminWeekFieldCallback обрабатывает настройку полей недели
func (b *Bot) handleAdminWeekFieldCallback(callbackQuery *tgbotapi.CallbackQuery, week int, field string) {
	userID := callbackQuery.From.ID

	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		return
	}

	var fieldName, example string

	switch field {
	case "title":
		fieldName = "Заголовок"
		example = "/setweek 1 title Неделя знакомства"
	case "welcome":
		fieldName = "Приветственное сообщение"
		example = "/setweek 1 welcome Добро пожаловать в первую неделю! Сегодня мы начинаем путь к более глубокому пониманию друг друга."
	case "questions":
		fieldName = "Упражнения"
		example = "/setweek 1 questions 1. Что вас больше всего привлекает в партнере? 2. Какие у вас общие цели?"
	case "tips":
		fieldName = "Подсказки"
		example = "/setweek 1 tips Будьте честными в своих ответах. Слушайте внимательно. Не бойтесь быть уязвимыми."
	case "insights":
		fieldName = "Инсайт"
		example = "/setweek 1 insights Понимание начинается с принятия различий друг друга."
	case "joint":
		fieldName = "Совместные вопросы"
		example = "/setweek 1 joint Обсудите вместе: Какие традиции вы хотели бы создать в ваших отношениях?"
	case "diary":
		fieldName = "Инструкции для дневника"
		example = "/setweek 1 diary Записывайте свои чувства после каждого упражнения. Что вы узнали о себе и партнере?"
	case "active":
		fieldName = "Активность недели"
		example = "/setweek 1 active true  (или false для закрытия)"
	default:
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Неизвестное поле")
		return
	}

	response := fmt.Sprintf("🗓️ Настройка: %s (%d неделя)\n\n"+
		"Используйте команду:\n"+
		"`/setweek %d %s <текст>`\n\n"+
		"Пример:\n"+
		"`%s`", fieldName, week, week, field, example)

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID

	switch message.Command() {
	case "start":
		response := b.welcomeMessage

		// Создаем простую inline клавиатуру с тремя основными функциями
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
		if b.isAdmin(userID) {
			adminRow := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👑 Админ-панель", "adminhelp"),
			)
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, adminRow)
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = keyboard
		b.telegram.Send(msg)

	case "chat":
		b.setUserState(userID, "chat")
		response := "💬 Режим обычной беседы активирован!\n\n" +
			"Теперь просто напишите мне любое сообщение, и я отвечу как обычный собеседник. " +
			"Я буду помнить нашу беседу и отвечать в контексте нашего разговора.\n\n" +
			"Чтобы получить упражнения на неделю, используйте /advice"
		b.sendMessage(message.Chat.ID, response)

	case "advice":
		response := "🗓️ Выберите неделю для упражнений:\n\n" +
			"Каждая неделя содержит специально подобранные упражнения для укрепления ваших отношений."

		// Создаем клавиатуру с выбором недель
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

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = weekKeyboard
		b.telegram.Send(msg)

	case "diary":
		// Получаем список активных недель
		activeWeeks := b.exercises.GetActiveWeeks()

		if len(activeWeeks) == 0 {
			response := "📝 Мини дневник\n\n" +
				"⚠️ В данный момент нет доступных недель для записей.\n" +
				"Администраторы еще не открыли доступ к неделям."
			b.sendMessage(message.Chat.ID, response)
			return
		}

		response := "📝 Мини дневник\n\n" +
			"Выберите доступную неделю для записи:"

		// Создаем кнопки только для активных недель
		var buttons [][]tgbotapi.InlineKeyboardButton
		var currentRow []tgbotapi.InlineKeyboardButton

		weekEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣"}

		for _, week := range activeWeeks {
			button := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s Неделя", weekEmojis[week-1]),
				fmt.Sprintf("diary_week_%d", week),
			)
			currentRow = append(currentRow, button)

			// Добавляем по 2 кнопки в ряд
			if len(currentRow) == 2 {
				buttons = append(buttons, currentRow)
				currentRow = []tgbotapi.InlineKeyboardButton{}
			}
		}

		// Добавляем оставшиеся кнопки
		if len(currentRow) > 0 {
			buttons = append(buttons, currentRow)
		}

		diaryKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = diaryKeyboard
		b.telegram.Send(msg)

	case "setprompt":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		// Получаем новый промпт из текста сообщения
		args := strings.SplitN(message.Text, " ", 2)
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			b.sendMessage(message.Chat.ID, "❌ Использование: /setprompt <новый промпт>\n\nПример:\n/setprompt Ты опытный психолог, который помогает людям с их проблемами.")
			return
		}

		newPrompt := strings.TrimSpace(args[1])
		b.systemPrompt = newPrompt

		response := fmt.Sprintf("✅ Системный промпт успешно изменен!\n\n🤖 Новый промпт:\n%s", newPrompt)
		b.sendMessage(message.Chat.ID, response)
		log.Printf("👑 Администратор %d изменил системный промпт", userID)

	case "setwelcome":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		// Получаем новое приветствие из текста сообщения
		args := strings.SplitN(message.Text, " ", 2)
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			b.sendMessage(message.Chat.ID, "❌ Использование: /setwelcome <новое приветствие>\n\nПример:\n/setwelcome Привет! 👋 Я Lovifyy Bot - ваш персональный помощник!")
			return
		}

		newWelcome := strings.TrimSpace(args[1])
		b.welcomeMessage = newWelcome

		response := fmt.Sprintf("✅ Приветственное сообщение успешно изменено!\n\n👋 Новое приветствие:\n%s", newWelcome)
		b.sendMessage(message.Chat.ID, response)
		log.Printf("👑 Администратор %d изменил приветственное сообщение", userID)

	case "setweek":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		// Парсим команду: /setweek <номер недели> <поле> <значение>
		args := strings.SplitN(message.Text, " ", 4)
		if len(args) < 4 {
			b.sendMessage(message.Chat.ID, "❌ Использование: /setweek <неделя> <поле> <значение>\n\nПоля: title, welcome, questions, tips, insights, joint, diary\n\nПример:\n/setweek 1 title Неделя знакомства")
			return
		}

		// Парсим номер недели
		week, err := strconv.Atoi(args[1])
		if err != nil || week < 1 || week > 4 {
			b.sendMessage(message.Chat.ID, "❌ Номер недели должен быть от 1 до 4")
			return
		}

		field := args[2]
		value := strings.TrimSpace(args[3])

		if value == "" {
			b.sendMessage(message.Chat.ID, "❌ Значение не может быть пустым")
			return
		}

		// Сохраняем поле
		err = b.exercises.SaveWeekField(week, field, value)
		if err != nil {
			log.Printf("Ошибка сохранения поля %s для недели %d: %v", field, week, err)
			b.sendMessage(message.Chat.ID, "❌ Ошибка сохранения: "+err.Error())
			return
		}

		var fieldName string
		switch field {
		case "title":
			fieldName = "Заголовок"
		case "welcome":
			fieldName = "Приветственное сообщение"
		case "questions":
			fieldName = "Упражнения"
		case "tips":
			fieldName = "Подсказки"
		case "insights":
			fieldName = "Инсайт"
		case "joint":
			fieldName = "Совместные вопросы"
		case "diary":
			fieldName = "Инструкции для дневника"
		default:
			fieldName = field
		}

		response := fmt.Sprintf("✅ %s для %d недели успешно сохранен!\n\n📝 %s:\n%s", fieldName, week, fieldName, value)
		b.sendMessage(message.Chat.ID, response)
		log.Printf("👑 Администратор %d настроил %s для недели %d", userID, field, week)

	case "adminhelp":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
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

		// Создаем админскую клавиатуру
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

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = adminKeyboard
		b.telegram.Send(msg)

	case "prompt":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		response := fmt.Sprintf("🤖 Текущий системный промпт:\n\n%s\n\n💡 Для изменения используйте:\n/setprompt <новый промпт>", b.systemPrompt)
		b.sendMessage(message.Chat.ID, response)

	case "setprompt_menu":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		response := "✏️ Изменение системного промпта\n\n" +
			"Отправьте команду в формате:\n" +
			"`/setprompt <новый промпт>`\n\n" +
			"💡 Готовые варианты:\n\n" +
			"Психолог:\n" +
			"`/setprompt Ты опытный психолог, который помогает людям с их личными проблемами. Будь сочувствующим и давай полезные советы.`\n\n" +
			"Дружелюбный помощник:\n" +
			"`/setprompt Ты дружелюбный помощник, готовый ответить на любые вопросы. Будь позитивным и полезным.`\n\n" +
			"Программист:\n" +
			"`/setprompt Ты программист-эксперт, специализирующийся на Go и веб-разработке. Помогай с кодом и объясняй концепции.`"
		b.sendMessage(message.Chat.ID, response)

	case "welcome":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		response := fmt.Sprintf("👋 Текущее приветственное сообщение:\n\n%s\n\n💡 Для изменения используйте:\n/setwelcome <новое приветствие>", b.welcomeMessage)
		b.sendMessage(message.Chat.ID, response)

	case "setwelcome_menu":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}

		response := "📝 Изменение приветственного сообщения\n\n" +
			"Отправьте команду в формате:\n" +
			"`/setwelcome <новое приветствие>`\n\n" +
			"💡 Готовые варианты:\n\n" +
			"Стандартное:\n" +
			"`/setwelcome Привет! 👋 Я Lovifyy Bot - ваш персональный помощник!`\n\n" +
			"Для пар:\n" +
			"`/setwelcome Добро пожаловать в Lovifyy Bot! ❤️ Я помогу укрепить ваши отношения через упражнения и дневник.`\n\n" +
			"Краткое:\n" +
			"`/setwelcome Привет! Выберите режим работы:`"
		b.sendMessage(message.Chat.ID, response)

	case "clear":
		// Очищаем историю пользователя
		err := b.history.ClearUserHistory(userID)
		if err != nil {
			log.Printf("Ошибка очистки истории для пользователя %d: %v", userID, err)
			b.sendMessage(message.Chat.ID, "❌ Ошибка при очистке истории")
			return
		}

		// Сбрасываем состояние пользователя
		b.setUserState(userID, "")

		response := "🗑️ История очищена!\n\n" +
			"Ваша история сообщений была полностью удалена. " +
			"Теперь мы можем начать общение с чистого листа!\n\n" +
			"Используйте /start для выбора режима работы."
		b.sendMessage(message.Chat.ID, response)

	case "help":
		response := "🤖 Справка по Lovifyy Bot:\n\n" +
			"💬 /chat - режим обычной беседы\n" +
			"🗓️ /advice - упражнения недели\n" +
			"📝 /diary - мини дневник\n" +
			"🗑️ /clear - очистить историю\n" +
			"🚀 /start - главное меню\n\n" +
			"Просто напишите мне сообщение для общения!"
		b.sendMessage(message.Chat.ID, response)

	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для главного меню.")
	}
}

// handleAIMessage обрабатывает сообщения через ИИ с учетом истории
func (b *Bot) handleAIMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}

	// Проверяем состояние пользователя
	userState := b.getUserState(userID)
	log.Printf("Состояние пользователя %d: '%s'", userID, userState)

	// Если пользователь в режиме дневника, сохраняем в отдельный файл дневника
	if strings.HasPrefix(userState, "diary_") {
		// Парсим состояние: diary_<gender>_<week>_<type>
		parts := strings.Split(userState, "_")
		if len(parts) >= 4 {
			gender := parts[1]
			week, err := strconv.Atoi(parts[2])
			if err == nil && week >= 1 && week <= 4 && (gender == "male" || gender == "female") {
				entryType := strings.Join(parts[3:], "_")

				// Сохраняем запись в дневник с указанием недели, типа и гендера
				err := b.history.SaveDiaryEntryWithGender(userID, username, message.Text, week, entryType, gender)
				if err != nil {
					log.Printf("Ошибка сохранения записи дневника: %v", err)
					b.sendMessage(message.Chat.ID, "❌ Ошибка сохранения записи в дневник")
					return
				}

				// Определяем тип записи для ответа
				var typeEmoji, typeName string
				switch entryType {
				case "questions":
					typeEmoji = "❓"
					typeName = "ответы на вопросы"
				case "joint":
					typeEmoji = "👫"
					typeName = "ответы на совместные вопросы"
				case "personal":
					typeEmoji = "💭"
					typeName = "личные записи"
				default:
					typeEmoji = "📝"
					typeName = "запись"
				}

				// Отправляем подтверждение
				diaryResponse := fmt.Sprintf("%s Запись сохранена в дневник (%d неделя - %s)\n\n"+
					"Можете продолжить писать записи этого типа или выберите другое действие через главное меню.", typeEmoji, week, typeName)
				b.sendMessage(message.Chat.ID, diaryResponse)

				// НЕ сбрасываем состояние - пользователь остается в режиме дневника
				return
			}
		}

		// Если не удалось распарсить состояние, сбрасываем его
		b.setUserState(userID, "chat")
	}

	// Обработка состояний уведомлений
	if userState == "notification_custom_date" {
		b.handleCustomDateInput(message)
		return
	}

	if strings.HasPrefix(userState, "notification_custom_time_") {
		dateStr := strings.TrimPrefix(userState, "notification_custom_time_")
		b.handleCustomTimeInput(message, dateStr)
		return
	}

	if strings.HasPrefix(userState, "notification_custom_") && strings.Contains(userState, "_") {
		parts := strings.Split(userState, "_")
		if len(parts) >= 4 && parts[0] == "notification" && parts[1] == "custom" {
			dateStr := parts[2]
			timeStr := parts[3]
			b.handleCustomNotificationTextInput(message, dateStr, timeStr)
			return
		}
	}

	if userState == "broadcast_custom" {
		b.handleCustomBroadcastInput(message)
		return
	}

	// Если состояние пустое (пользователь еще не выбрал режим), показываем главное меню
	if userState == "" {
		response := "Привет! 👋 Выберите режим работы:"

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

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = keyboard
		b.telegram.Send(msg)
		return
	}

	// Если пользователь НЕ в режиме чата, показываем главное меню
	if userState != "chat" {
		response := "Выберите режим работы:"

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

		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = keyboard
		b.telegram.Send(msg)
		return
	}

	// Отправляем индикатор печати для режима чата
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Получаем историю в формате OpenAI (последние 10 сообщений)
	openaiMessages, err := b.history.GetOpenAIHistory(userID, b.systemPrompt, 10)
	if err != nil {
		log.Printf("Ошибка получения истории: %v", err)
		openaiMessages = []history.OpenAIMessage{
			{Role: "system", Content: b.systemPrompt},
		}
	}

	log.Printf("Загружено %d сообщений из истории для пользователя %d", len(openaiMessages), userID)

	// Добавляем новое сообщение пользователя
	openaiMessages = append(openaiMessages, history.OpenAIMessage{
		Role:    "user",
		Content: message.Text,
	})

	// Конвертируем в формат AI клиента
	aiMessages := make([]ai.OpenAIMessage, len(openaiMessages))
	for i, msg := range openaiMessages {
		aiMessages[i] = ai.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Получаем ответ от OpenAI с полной историей
	response, err := b.ai.GenerateWithHistory(aiMessages)
	if err != nil {
		log.Printf("Ошибка получения ответа от ИИ: %v", err)
		b.sendMessage(message.Chat.ID, "Извините, произошла ошибка при обработке вашего сообщения. Попробуйте еще раз.")
		return
	}

	// Очищаем ответ
	response = strings.TrimSpace(response)

	// Сохраняем в историю
	err = b.history.SaveMessage(userID, username, message.Text, response, "gpt-4o-mini")
	if err != nil {
		log.Printf("Ошибка сохранения в историю: %v", err)
	}

	// Отправляем ответ пользователю
	b.sendMessage(message.Chat.ID, response)
}

// sendMessage отправляет сообщение пользователю
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := b.telegram.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// generatePersonalInsight генерирует персональный инсайт на основе истории пользователя
func (b *Bot) generatePersonalInsight(callbackQuery *tgbotapi.CallbackQuery, week int) {
	userID := callbackQuery.From.ID
	username := callbackQuery.From.UserName
	if username == "" {
		username = callbackQuery.From.FirstName
	}

	// Отправляем индикатор печати
	typing := tgbotapi.NewChatAction(callbackQuery.Message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Получаем записи дневника для конкретной недели
	diaryEntries, err := b.history.GetDiaryEntriesByWeek(userID, week)
	if err != nil || len(diaryEntries) == 0 {
		// Если нет записей в дневнике для этой недели, показываем сообщение
		response := fmt.Sprintf("🔍 Персональный инсайт (%d неделя)\n\n"+
			"Для создания персонального инсайта для %d недели мне нужны ваши записи в дневнике. "+
			"Сначала сделайте записи в дневнике для этой недели, а затем вернитесь к инсайту.\n\n"+
			"📝 Используйте кнопку \"Что писать в дневнике\" для получения инструкций", week, week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Формируем контекст из записей дневника
	var diaryContext string
	for _, entry := range diaryEntries {
		var entryTypeName string
		switch entry.Type {
		case "questions":
			entryTypeName = "Ответы на упражнения"
		case "joint":
			entryTypeName = "Совместные вопросы"
		case "personal":
			entryTypeName = "Личные записи"
		default:
			entryTypeName = "Запись"
		}
		diaryContext += fmt.Sprintf("%s: %s\n\n", entryTypeName, entry.Entry)
	}

	// Создаем сообщения для OpenAI
	openaiMessages := []history.OpenAIMessage{
		{
			Role:    "system",
			Content: b.systemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Вот мои записи из дневника за %d неделю:\n\n%s", week, diaryContext),
		},
	}

	// Добавляем специальный запрос для генерации инсайта
	insightPrompt := "После анализа нашего разговора составь краткое резюме в следующем формате:\n\n" +
		"«Судя по вашим ответам, вы цените [качества] и чаще всего испытываете [чувство/тревогу] в ситуациях, когда [описание ситуации]. Обсудите вместе, как это влияет на ваши отношения».\n\n" +
		"Проанализируй нашу беседу и дай персональный инсайт именно в этом формате."

	openaiMessages = append(openaiMessages, history.OpenAIMessage{
		Role:    "user",
		Content: insightPrompt,
	})

	// Конвертируем в формат AI клиента
	aiMessages := make([]ai.OpenAIMessage, len(openaiMessages))
	for i, msg := range openaiMessages {
		aiMessages[i] = ai.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Получаем инсайт от OpenAI
	insightResponse, err := b.ai.GenerateWithHistory(aiMessages)
	if err != nil {
		log.Printf("Ошибка генерации инсайта: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка при генерации персонального инсайта. Попробуйте позже.")
		return
	}

	// Формируем финальный ответ
	response := fmt.Sprintf("🔍 Персональный инсайт (%d неделя)\n\n%s", week, strings.TrimSpace(insightResponse))

	// Сохраняем в историю
	err = b.history.SaveMessage(userID, username, "Запрос персонального инсайта", insightResponse, "gpt-4o-mini")
	if err != nil {
		log.Printf("Ошибка сохранения инсайта в историю: %v", err)
	}

	// Отправляем инсайт пользователю
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleDiaryViewCallback обрабатывает нажатие кнопки "Посмотреть свои записи"
func (b *Bot) handleDiaryViewCallback(callbackQuery *tgbotapi.CallbackQuery) {
	// Получаем список активных недель
	activeWeeks := b.exercises.GetActiveWeeks()

	if len(activeWeeks) == 0 {
		response := "👀 Просмотр записей\n\n" +
			"⚠️ В данный момент нет доступных недель для просмотра записей.\n" +
			"Администраторы еще не открыли доступ к неделям."
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	response := "👀 Просмотр записей дневника\n\n" +
		"Сначала выберите, чьи записи хотите посмотреть:"

	// Создаем кнопки выбора гендера для просмотра
	buttons := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Записи парня", "diary_view_gender_male"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Записи девушки", "diary_view_gender_female"),
		),
	}

	viewKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = viewKeyboard
	b.telegram.Send(msg)
}

// handleDiaryViewWeekCallback обрабатывает просмотр записей конкретной недели
func (b *Bot) handleDiaryViewWeekCallback(callbackQuery *tgbotapi.CallbackQuery, week int) {
	userID := callbackQuery.From.ID

	// Получаем все записи пользователя для этой недели из всех типов
	var allEntries []history.DiaryEntry

	// Читаем из всех типов дневников
	typeDirs := []string{"diary_questions", "diary_jointquestions", "diary_thoughts"}
	typeNames := map[string]string{
		"diary_questions":      "💪 Ответы на упражнения",
		"diary_jointquestions": "👫 Совместные вопросы",
		"diary_thoughts":       "💭 Личные мысли",
	}

	for _, typeDir := range typeDirs {
		entries, err := b.getDiaryEntriesByTypeAndWeek(userID, typeDir, week)
		if err == nil {
			allEntries = append(allEntries, entries...)
		}
	}

	// Также проверяем старые файлы для совместимости
	oldEntries, err := b.history.GetDiaryEntriesByWeek(userID, week)
	if err == nil {
		allEntries = append(allEntries, oldEntries...)
	}

	if len(allEntries) == 0 {
		response := fmt.Sprintf("👀 Записи за %d неделю\n\n"+
			"📝 У вас пока нет записей за эту неделю.\n"+
			"Начните писать дневник, чтобы увидеть здесь свои записи!", week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Группируем записи по типам
	entriesByType := make(map[string][]history.DiaryEntry)
	for _, entry := range allEntries {
		entriesByType[entry.Type] = append(entriesByType[entry.Type], entry)
	}

	// Формируем ответ
	response := fmt.Sprintf("👀 Ваши записи за %d неделю\n\n", week)

	for entryType, entries := range entriesByType {
		typeName := typeNames["diary_"+entryType]
		if typeName == "" {
			switch entryType {
			case "questions":
				typeName = "💪 Ответы на упражнения"
			case "joint":
				typeName = "👫 Совместные вопросы"
			case "personal":
				typeName = "💭 Личные мысли"
			default:
				typeName = "📝 Записи"
			}
		}

		response += fmt.Sprintf("%s:\n", typeName)
		for i, entry := range entries {
			// Обрезаем длинные записи для краткого просмотра
			entryText := entry.Entry
			if len(entryText) > 200 {
				entryText = entryText[:200] + "..."
			}
			response += fmt.Sprintf("%d. %s\n", i+1, entryText)
		}
		response += "\n"
	}

	response += "💡 *Для добавления новых записей используйте основное меню дневника*"

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// getDiaryEntriesByTypeAndWeek получает записи дневника конкретного типа и недели
func (b *Bot) getDiaryEntriesByTypeAndWeek(userID int64, typeDir string, week int) ([]history.DiaryEntry, error) {
	filename := filepath.Join("diary_entries", typeDir, fmt.Sprintf("user_%d.json", userID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return []history.DiaryEntry{}, nil // Возвращаем пустой массив если файла нет
	}

	var entries []history.DiaryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	// Фильтруем по неделе
	var weekEntries []history.DiaryEntry
	for _, entry := range entries {
		if entry.Week == week {
			weekEntries = append(weekEntries, entry)
		}
	}

	return weekEntries, nil
}

// handleDiaryGenderCallback обрабатывает выбор гендера для дневника
func (b *Bot) handleDiaryGenderCallback(callbackQuery *tgbotapi.CallbackQuery, gender string) {
	// Получаем список активных недель
	activeWeeks := b.exercises.GetActiveWeeks()

	if len(activeWeeks) == 0 {
		genderName := "парня"
		if gender == "female" {
			genderName = "девушки"
		}
		response := fmt.Sprintf("📝 Дневник для %s\n\n"+
			"⚠️ В данный момент нет доступных недель для записей.\n"+
			"Администраторы еще не открыли доступ к неделям.", genderName)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}

	response := fmt.Sprintf("%s Дневник для %s\n\n"+
		"Выберите доступную неделю для записи:", genderEmoji, genderName)

	// Создаем кнопки только для активных недель
	var buttons [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	weekEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣"}

	for _, week := range activeWeeks {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s Неделя %d", weekEmojis[week-1], week),
			fmt.Sprintf("diary_week_%s_%d", gender, week),
		)
		currentRow = append(currentRow, button)

		// Добавляем по 2 кнопки в ряд
		if len(currentRow) == 2 {
			buttons = append(buttons, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Добавляем оставшиеся кнопки
	if len(currentRow) > 0 {
		buttons = append(buttons, currentRow)
	}

	diaryKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = diaryKeyboard
	b.telegram.Send(msg)
}

// handleDiaryWeekGenderCallback обрабатывает выбор недели для дневника с гендером
func (b *Bot) handleDiaryWeekGenderCallback(callbackQuery *tgbotapi.CallbackQuery, gender string, week int) {
	// Проверяем, активна ли неделя
	if !b.exercises.IsWeekActive(week) {
		response := fmt.Sprintf("❌ Неделя %d недоступна\n\n"+
			"Эта неделя еще не открыта администраторами.", week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}

	response := fmt.Sprintf("%s Дневник для %s - %d неделя\n\n"+
		"Выберите тип записи:", genderEmoji, genderName, week)

	// Создаем кнопки для типов записей
	typeKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💪 Ответы на упражнения", fmt.Sprintf("diary_%s_%d_type_questions", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Совместные вопросы", fmt.Sprintf("diary_%s_%d_type_joint", gender, week)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💭 Личные мысли", fmt.Sprintf("diary_%s_%d_type_personal", gender, week)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = typeKeyboard
	b.telegram.Send(msg)
}

// handleDiaryTypeGenderCallback обрабатывает выбор типа записи с гендером
func (b *Bot) handleDiaryTypeGenderCallback(callbackQuery *tgbotapi.CallbackQuery, gender string, week int, entryType string) {
	userID := callbackQuery.From.ID

	// Устанавливаем состояние пользователя для дневника с гендером
	b.setUserState(userID, fmt.Sprintf("diary_%s_%d_%s", gender, week, entryType))

	var response string
	var typeName string
	genderName := "парня"
	if gender == "female" {
		genderName = "девушки"
	}

	switch entryType {
	case "questions":
		typeName = "ответы на упражнения"
		response = fmt.Sprintf("💪 Ответы на упражнения для %s (%d неделя)\n\n"+
			"Напишите ваши ответы на упражнения этой недели. "+
			"Будьте честными и открытыми в своих ответах.", genderName, week)
	case "joint":
		typeName = "совместные вопросы"
		response = fmt.Sprintf("👫 Совместные вопросы для %s (%d неделя)\n\n"+
			"Напишите ваши ответы на совместные вопросы. "+
			"Эти записи помогут вам лучше понять друг друга.", genderName, week)
	case "personal":
		typeName = "личные мысли"
		response = fmt.Sprintf("💭 Личные мысли для %s (%d неделя)\n\n"+
			"Поделитесь своими личными мыслями и переживаниями. "+
			"Это пространство только для ваших размышлений.", genderName, week)
	default:
		typeName = "записи"
		response = fmt.Sprintf("📝 Записи для %s (%d неделя)\n\n"+
			"Напишите ваши мысли и наблюдения.", genderName, week)
	}

	log.Printf("Пользователь %d начал запись в дневник: %s, неделя %d, тип %s, гендер %s",
		userID, typeName, week, entryType, gender)

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleDiaryViewGenderCallback обрабатывает выбор гендера для просмотра записей
func (b *Bot) handleDiaryViewGenderCallback(callbackQuery *tgbotapi.CallbackQuery, gender string) {
	// Получаем список активных недель
	activeWeeks := b.exercises.GetActiveWeeks()

	if len(activeWeeks) == 0 {
		genderName := "парня"
		if gender == "female" {
			genderName = "девушки"
		}
		response := fmt.Sprintf("👀 Записи %s\n\n"+
			"⚠️ В данный момент нет доступных недель для просмотра записей.\n"+
			"Администраторы еще не открыли доступ к неделям.", genderName)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}

	response := fmt.Sprintf("%s Записи %s\n\n"+
		"Выберите неделю для просмотра записей:", genderEmoji, genderName)

	// Создаем кнопки только для активных недель
	var buttons [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	weekEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣"}

	for _, week := range activeWeeks {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s Неделя %d", weekEmojis[week-1], week),
			fmt.Sprintf("diary_view_week_%s_%d", gender, week),
		)
		currentRow = append(currentRow, button)

		// Добавляем по 2 кнопки в ряд
		if len(currentRow) == 2 {
			buttons = append(buttons, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Добавляем оставшиеся кнопки
	if len(currentRow) > 0 {
		buttons = append(buttons, currentRow)
	}

	viewKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = viewKeyboard
	b.telegram.Send(msg)
}

// handleDiaryViewWeekGenderCallback обрабатывает просмотр записей конкретной недели с гендером
func (b *Bot) handleDiaryViewWeekGenderCallback(callbackQuery *tgbotapi.CallbackQuery, gender string, week int) {
	userID := callbackQuery.From.ID

	// Получаем все записи пользователя для этой недели из всех типов с учетом гендера
	var allEntries []history.DiaryEntry

	// Читаем из всех типов дневников с гендером
	typeDirs := []string{"diary_questions", "diary_jointquestions", "diary_thoughts"}
	typeNames := map[string]string{
		"diary_questions":      "💪 Ответы на упражнения",
		"diary_jointquestions": "👫 Совместные вопросы",
		"diary_thoughts":       "💭 Личные мысли",
	}

	for _, typeDir := range typeDirs {
		entries, err := b.getDiaryEntriesByTypeWeekAndGender(userID, typeDir, week, gender)
		if err == nil {
			allEntries = append(allEntries, entries...)
		}
	}

	// Также проверяем старые файлы для совместимости
	oldEntries, err := b.history.GetDiaryEntriesByWeek(userID, week)
	if err == nil {
		allEntries = append(allEntries, oldEntries...)
	}

	genderName := "парня"
	genderEmoji := "👨"
	if gender == "female" {
		genderName = "девушки"
		genderEmoji = "👩"
	}

	if len(allEntries) == 0 {
		response := fmt.Sprintf("%s Записи %s за %d неделю\n\n"+
			"📝 Пока нет записей за эту неделю.\n"+
			"Начните писать дневник, чтобы увидеть здесь записи!", genderEmoji, genderName, week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Группируем записи по типам
	entriesByType := make(map[string][]history.DiaryEntry)
	for _, entry := range allEntries {
		entriesByType[entry.Type] = append(entriesByType[entry.Type], entry)
	}

	// Формируем ответ
	response := fmt.Sprintf("%s Записи %s за %d неделю\n\n", genderEmoji, genderName, week)

	for entryType, entries := range entriesByType {
		typeName := typeNames["diary_"+entryType]
		if typeName == "" {
			switch entryType {
			case "questions":
				typeName = "💪 Ответы на упражнения"
			case "joint":
				typeName = "👫 Совместные вопросы"
			case "personal":
				typeName = "💭 Личные мысли"
			default:
				typeName = "📝 Записи"
			}
		}

		response += fmt.Sprintf("%s:\n", typeName)
		for i, entry := range entries {
			// Обрезаем длинные записи для краткого просмотра
			entryText := entry.Entry
			if len(entryText) > 200 {
				entryText = entryText[:200] + "..."
			}
			response += fmt.Sprintf("%d. %s\n", i+1, entryText)
		}
		response += "\n"
	}

	response += "💡 *Для добавления новых записей используйте основное меню дневника*"

	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// getDiaryEntriesByTypeWeekAndGender получает записи дневника конкретного типа, недели и гендера
func (b *Bot) getDiaryEntriesByTypeWeekAndGender(userID int64, typeDir string, week int, gender string) ([]history.DiaryEntry, error) {
	// Новая структура: diary_entries/typeDir/week/gender/user_ID.json
	filename := filepath.Join("diary_entries", typeDir, fmt.Sprintf("%d", week), gender, fmt.Sprintf("user_%d.json", userID))

	data, err := os.ReadFile(filename)
	if err != nil {
		// Пробуем старую структуру для совместимости: diary_entries/typeDir/gender/user_ID.json
		oldFilename := filepath.Join("diary_entries", typeDir, gender, fmt.Sprintf("user_%d.json", userID))
		data, err = os.ReadFile(oldFilename)
		if err != nil {
			return []history.DiaryEntry{}, nil // Возвращаем пустой массив если файла нет
		}
		
		// Если читаем из старого файла, фильтруем по неделе
		var entries []history.DiaryEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, err
		}

		var weekEntries []history.DiaryEntry
		for _, entry := range entries {
			if entry.Week == week {
				weekEntries = append(weekEntries, entry)
			}
		}
		return weekEntries, nil
	}

	// Читаем из новой структуры - все записи уже для нужной недели
	var entries []history.DiaryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// handleInsightGenderChoice показывает выбор гендера для генерации инсайта
func (b *Bot) handleInsightGenderChoice(callbackQuery *tgbotapi.CallbackQuery, week int) {
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
	b.telegram.Send(msg)
}

// generatePersonalInsightWithGender генерирует персональный инсайт с учетом гендера
func (b *Bot) generatePersonalInsightWithGender(callbackQuery *tgbotapi.CallbackQuery, gender string, week int) {
	userID := callbackQuery.From.ID
	username := callbackQuery.From.UserName
	if username == "" {
		username = callbackQuery.From.FirstName
	}

	// Отправляем индикатор печати
	typing := tgbotapi.NewChatAction(callbackQuery.Message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Получаем записи дневника для конкретной недели с учетом гендера
	diaryEntries, err := b.getDiaryEntriesByWeekAndGender(userID, week, gender)
	if err != nil || len(diaryEntries) == 0 {
		genderName := "парня"
		if gender == "female" {
			genderName = "девушки"
		}
		// Если нет записей в дневнике для этой недели, показываем сообщение
		response := fmt.Sprintf("🔍 Персональный инсайт для %s (%d неделя)\n\n"+
			"Для создания персонального инсайта для %s в %d неделе мне нужны записи в дневнике. "+
			"Сначала сделайте записи в дневнике для этой недели, а затем вернитесь к инсайту.\n\n"+
			"📝 Используйте кнопку \"Мини дневник\" для записи мыслей", genderName, genderName, week)
		b.sendMessage(callbackQuery.Message.Chat.ID, response)
		return
	}

	// Формируем контекст из записей дневника
	var diaryContext string
	for _, entry := range diaryEntries {
		var entryTypeName string
		switch entry.Type {
		case "questions":
			entryTypeName = "Ответы на упражнения"
		case "joint":
			entryTypeName = "Совместные вопросы"
		case "personal":
			entryTypeName = "Личные записи"
		default:
			entryTypeName = "Запись"
		}
		diaryContext += fmt.Sprintf("%s: %s\n\n", entryTypeName, entry.Entry)
	}

	genderName := "парня"
	if gender == "female" {
		genderName = "девушки"
	}

	// Создаем сообщения для OpenAI
	openaiMessages := []history.OpenAIMessage{
		{
			Role:    "system",
			Content: b.systemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Вот записи из дневника для %s за %d неделю:\n\n%s", genderName, week, diaryContext),
		},
	}

	// Добавляем специальный запрос для генерации инсайта
	insightPrompt := "После анализа записей составь краткое резюме в следующем формате:\n\n" +
		"«Судя по записям, вы цените [качества] и чаще всего испытываете [чувство/тревогу] в ситуациях, когда [описание ситуации]. Обсудите вместе, как это влияет на ваши отношения».\n\n" +
		"Проанализируй записи и дай персональный инсайт именно в этом формате."

	openaiMessages = append(openaiMessages, history.OpenAIMessage{
		Role:    "user",
		Content: insightPrompt,
	})

	// Конвертируем в формат AI клиента
	aiMessages := make([]ai.OpenAIMessage, len(openaiMessages))
	for i, msg := range openaiMessages {
		aiMessages[i] = ai.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Получаем инсайт от OpenAI
	insightResponse, err := b.ai.GenerateWithHistory(aiMessages)
	if err != nil {
		log.Printf("Ошибка генерации инсайта: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Ошибка при генерации персонального инсайта. Попробуйте позже.")
		return
	}

	// Формируем финальный ответ
	response := fmt.Sprintf("🔍 Персональный инсайт для %s (%d неделя)\n\n%s", genderName, week, strings.TrimSpace(insightResponse))

	// Сохраняем в историю
	err = b.history.SaveMessage(userID, username, fmt.Sprintf("Запрос персонального инсайта для %s", genderName), insightResponse, "gpt-4o-mini")
	if err != nil {
		log.Printf("Ошибка сохранения инсайта в историю: %v", err)
	}

	// Отправляем инсайт пользователю
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// getDiaryEntriesByWeekAndGender получает записи дневника для недели с учетом гендера
func (b *Bot) getDiaryEntriesByWeekAndGender(userID int64, week int, gender string) ([]history.DiaryEntry, error) {
	var allWeekEntries []history.DiaryEntry

	// Читаем записи из папки "diary_questions" с гендером
	questionsEntries, err := b.getDiaryEntriesByTypeWeekAndGender(userID, "diary_questions", week, gender)
	if err == nil {
		allWeekEntries = append(allWeekEntries, questionsEntries...)
	}

	// Читаем записи из папки "diary_thoughts" с гендером
	thoughtsEntries, err := b.getDiaryEntriesByTypeWeekAndGender(userID, "diary_thoughts", week, gender)
	if err == nil {
		allWeekEntries = append(allWeekEntries, thoughtsEntries...)
	}

	// Для совместимости со старыми записями - читаем из старых файлов
	oldEntries, err := b.history.GetDiaryEntriesByWeek(userID, week)
	if err == nil {
		for _, entry := range oldEntries {
			if entry.Type == "questions" || entry.Type == "personal" {
				allWeekEntries = append(allWeekEntries, entry)
			}
		}
	}

	return allWeekEntries, nil
}

// handleCustomDateInput обрабатывает ввод кастомной даты
func (b *Bot) handleCustomDateInput(message *tgbotapi.Message) {
	userID := message.From.ID
	dateStr := strings.TrimSpace(message.Text)

	// Проверяем формат даты (ДД.ММ.ГГГГ)
	_, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		response := "❌ Неверный формат даты!\n\n" +
			"Используйте формат ДД.ММ.ГГГГ\n" +
			"Например: 15.10.2025"
		b.sendMessage(message.Chat.ID, response)
		return
	}

	// Сбрасываем состояние и переходим к выбору времени
	b.setUserState(userID, "")
	b.handleScheduleDateCallback(&tgbotapi.CallbackQuery{
		From:    message.From,
		Message: message,
	}, dateStr)
}

// handleCustomTimeInput обрабатывает ввод кастомного времени
func (b *Bot) handleCustomTimeInput(message *tgbotapi.Message, dateStr string) {
	userID := message.From.ID
	timeStr := strings.TrimSpace(message.Text)

	// Проверяем формат времени (ЧЧ:ММ)
	_, err := time.Parse("15:04", timeStr)
	if err != nil {
		response := "❌ Неверный формат времени!\n\n" +
			"Используйте формат ЧЧ:ММ\n" +
			"Например: 15:30"
		b.sendMessage(message.Chat.ID, response)
		return
	}

	// Сбрасываем состояние и переходим к выбору шаблона
	b.setUserState(userID, "")
	b.handleScheduleTimeCallback(&tgbotapi.CallbackQuery{
		From:    message.From,
		Message: message,
	}, dateStr, timeStr)
}

// handleCustomNotificationTextInput обрабатывает ввод кастомного текста уведомления
func (b *Bot) handleCustomNotificationTextInput(message *tgbotapi.Message, dateStr, timeStr string) {
	userID := message.From.ID
	messageText := strings.TrimSpace(message.Text)

	if len(messageText) == 0 {
		response := "❌ Текст уведомления не может быть пустым!"
		b.sendMessage(message.Chat.ID, response)
		return
	}

	// Сбрасываем состояние
	b.setUserState(userID, "")

	// Сохраняем уведомление в файл
	if err := b.saveNotification(dateStr, timeStr, messageText); err != nil {
		log.Printf("Ошибка сохранения уведомления: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка сохранения уведомления")
		return
	}

	response := fmt.Sprintf("✅ Уведомление запланировано!\n\n📅 Дата: %s\n🕐 Время: %s (UTC+5)\n\n💌 Текст:\n%s\n\n" +
		"⚠️ Уведомление будет отправлено всем пользователям бота", dateStr, timeStr, messageText)
	
	log.Printf("👑 Администратор %d запланировал уведомление на %s %s: %s", userID, dateStr, timeStr, messageText)
	b.sendMessage(message.Chat.ID, response)
}

// handleCustomBroadcastInput обрабатывает ввод кастомного текста для мгновенной отправки
func (b *Bot) handleCustomBroadcastInput(message *tgbotapi.Message) {
	userID := message.From.ID
	messageText := strings.TrimSpace(message.Text)

	if len(messageText) == 0 {
		response := "❌ Текст сообщения не может быть пустым!"
		b.sendMessage(message.Chat.ID, response)
		return
	}

	// Сбрасываем состояние
	b.setUserState(userID, "")

	// Отправляем уведомление всем пользователям
	sentCount, err := b.broadcastMessage(messageText)
	if err != nil {
		log.Printf("Ошибка отправки уведомления: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Ошибка отправки уведомления")
		return
	}

	response := fmt.Sprintf("✅ Уведомление отправлено!\n\n💌 Текст:\n%s\n\n" +
		"📤 Сообщение отправлено %d пользователям", messageText, sentCount)
	
	log.Printf("👑 Администратор %d отправил кастомное уведомление %d пользователям", userID, sentCount)
	log.Printf("📝 Полный текст кастомного уведомления: %s", messageText)
	b.sendMessage(message.Chat.ID, response)
}

// cleanUTF8 очищает строку от невалидных UTF-8 символов
func (b *Bot) cleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	
	// Если строка содержит невалидные UTF-8 символы, очищаем её
	cleaned := strings.ToValidUTF8(s, "")
	// Дополнительно убираем пустые символы в конце
	cleaned = strings.TrimRight(cleaned, "\x00\uFFFD")
	return cleaned
}

// generateNotificationTemplate генерирует динамический шаблон уведомления с помощью GPT
func (b *Bot) generateNotificationTemplate(templateType string) string {
	var prompt string
	
	switch templateType {
	case "diary":
		prompt = "Создай короткое мотивирующее сообщение (до 100 символов) о важности ведения дневника отношений. Используй теплый тон и простые эмодзи. Начни с 'Привет!'"
	case "exercises":
		prompt = "Создай короткое сообщение (до 100 символов) о пользе психологических упражнений и заданий для укрепления отношений пар. Используй мотивирующий тон и простые эмодзи. Упомяни 'упражнения для отношений' или 'задания для пар'"
	case "motivation":
		prompt = "Создай короткую мотивирующую цитату (до 100 символов) об отношениях и любви. Используй вдохновляющий тон и простые эмодзи"
	default:
		return "Привет! ❤️ Напоминание от вашего бота о важности отношений!"
	}
	
	// Генерируем ответ через AI
	log.Printf("🤖 Генерируем шаблон типа '%s' с промптом: %s", templateType, prompt)
	startTime := time.Now()
	response, err := b.ai.Generate(prompt)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("❌ Ошибка генерации шаблона %s: %v", templateType, err)
		// Возвращаем запасной вариант при ошибке
		switch templateType {
		case "diary":
			return "Привет! ❤️ Время заполнить дневник - ваши мысли важны!"
		case "exercises":
			return "Время для упражнений для отношений! 💑 Психологические задания помогут вам стать ближе!"
		case "motivation":
			return "Каждый день - шанс стать ближе! 🌟 Цените моменты вместе!"
		default:
			return "Привет! ❤️ Напоминание от вашего бота!"
		}
	}
	
	// Очищаем ответ от лишних символов и проблем с UTF-8
	cleanResponse := strings.TrimSpace(response)
	cleanResponse = b.cleanUTF8(cleanResponse)
	
	// Логируем полный ответ с временем генерации
	log.Printf("✅ Сгенерирован шаблон '%s' за %.2f сек (длина %d): %s", templateType, duration.Seconds(), len(cleanResponse), cleanResponse)
	
	// Возвращаем полный текст без ограничений
	return cleanResponse
}

// broadcastMessage отправляет сообщение всем пользователям бота
func (b *Bot) broadcastMessage(messageText string) (int, error) {
	// Получаем список всех пользователей из истории чатов
	userIDs, err := b.getAllUserIDs()
	if err != nil {
		return 0, err
	}

	sentCount := 0
	for _, userID := range userIDs {
		// Отправляем сообщение каждому пользователю
		msg := tgbotapi.NewMessage(userID, messageText)
		_, err := b.telegram.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения пользователю %d: %v", userID, err)
			continue // Продолжаем отправку другим пользователям
		}
		sentCount++
		
		// Небольшая задержка чтобы не превысить лимиты API
		time.Sleep(50 * time.Millisecond)
	}

	return sentCount, nil
}

// getAllUserIDs получает список всех пользователей из файлов истории
func (b *Bot) getAllUserIDs() ([]int64, error) {
	userIDsMap := make(map[int64]bool)
	
	// Читаем папку chat_history
	files, err := os.ReadDir("chat_history")
	if err != nil {
		if os.IsNotExist(err) {
			return []int64{}, nil
		}
		return nil, err
	}

	// Извлекаем ID пользователей из имен файлов
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "user_") && strings.HasSuffix(file.Name(), ".json") {
			// Извлекаем ID из имени файла: user_123456.json
			idStr := strings.TrimPrefix(file.Name(), "user_")
			idStr = strings.TrimSuffix(idStr, ".json")
			
			if userID, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				userIDsMap[userID] = true
			}
		}
	}

	// Также читаем папки дневников для получения дополнительных пользователей
	diaryDirs := []string{"diary_entries/diary_questions", "diary_entries/diary_jointquestions", "diary_entries/diary_thoughts"}
	for _, dir := range diaryDirs {
		b.addUsersFromDiaryDir(dir, userIDsMap)
	}

	// Конвертируем map в slice
	var userIDs []int64
	for userID := range userIDsMap {
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

// addUsersFromDiaryDir добавляет пользователей из папки дневника
func (b *Bot) addUsersFromDiaryDir(dir string, userIDsMap map[int64]bool) {
	// Проходим по всем подпапкам (недели и гендеры)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if strings.HasPrefix(info.Name(), "user_") && strings.HasSuffix(info.Name(), ".json") {
			// Извлекаем ID из имени файла: user_123456.json
			idStr := strings.TrimPrefix(info.Name(), "user_")
			idStr = strings.TrimSuffix(idStr, ".json")
			
			if userID, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				userIDsMap[userID] = true
			}
		}
		return nil
	})
}

// StartNotificationScheduler запускает планировщик уведомлений
func (b *Bot) StartNotificationScheduler() {
	log.Println("⏰ Планировщик уведомлений запущен")
	
	// Проверяем уведомления каждую минуту
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.checkAndSendScheduledNotifications()
		}
	}
}

// checkAndSendScheduledNotifications проверяет и отправляет запланированные уведомления
func (b *Bot) checkAndSendScheduledNotifications() {
	notifications, err := b.loadScheduledNotifications()
	if err != nil {
		log.Printf("Ошибка загрузки уведомлений: %v", err)
		return
	}

	// Используем UTC+5 вручную
	location := time.FixedZone("UTC+5", 5*60*60)
	now := time.Now().In(location)
	currentDate := now.Format("02.01.2006")
	currentTime := now.Format("15:04")
	
	log.Printf("🕐 Проверка уведомлений: %s %s (UTC+5)", currentDate, currentTime)

	var remainingNotifications []ScheduledNotification

	for _, notification := range notifications {
		// Парсим время уведомления и добавляем 5 часов для UTC
		notificationTime, err := time.Parse("15:04", notification.Time)
		if err != nil {
			log.Printf("Ошибка парсинга времени %s: %v", notification.Time, err)
			remainingNotifications = append(remainingNotifications, notification)
			continue
		}
		
		// Отнимаем 5 часов от времени уведомления для конвертации в UTC
		utcTime := notificationTime.Add(-5 * time.Hour)
		utcTimeStr := utcTime.Format("15:04")
		
		// Получаем текущее UTC время
		nowUTC := time.Now().UTC()
		currentUTCTime := nowUTC.Format("15:04")
		
		// Проверяем, пришло ли время отправки (сравниваем дату с UTC+5, время с UTC)
		if notification.Date == currentDate && utcTimeStr == currentUTCTime {
			log.Printf("⏰ Отправляем запланированное уведомление ID %d (UTC+5: %s %s -> UTC: %s %s)", 
				notification.ID, notification.Date, notification.Time, currentDate, utcTimeStr)
			
			// Отправляем уведомление
			sentCount, err := b.broadcastMessage(notification.Message)
			if err != nil {
				log.Printf("Ошибка отправки запланированного уведомления ID %d: %v", notification.ID, err)
			} else {
				log.Printf("✅ Запланированное уведомление ID %d отправлено %d пользователям", notification.ID, sentCount)
			}
			
			// Не добавляем в remainingNotifications - уведомление отправлено и удаляется
		} else {
			// Оставляем уведомление для будущей отправки
			remainingNotifications = append(remainingNotifications, notification)
		}
	}

	// Сохраняем обновленный список уведомлений (без отправленных)
	if len(remainingNotifications) != len(notifications) {
		err := b.saveScheduledNotifications(remainingNotifications)
		if err != nil {
			log.Printf("Ошибка сохранения обновленного списка уведомлений: %v", err)
		}
	}
}
