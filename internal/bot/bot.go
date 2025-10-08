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
		{Command: "chat", Description: "💬 Обычная беседа"},
		{Command: "advice", Description: "🗓️ Упражнения недели"},
		{Command: "diary", Description: "📝 Мини дневник"},
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
	defaultWelcome := "Привет! 👋 Я Lovifyy Bot - ваш персональный помощник!\n\n" +
		"🤖 Работаю полностью локально с ИИ\n" +
		"💾 Запоминаю всю нашу беседу\n" +
		"🗓️ Готов подготовить упражнения на неделю на основе нашего общения\n\n" +
		"Выберите режим работы:"

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
	case "exercise_week_1":
		b.handleExerciseWeekCallback(callbackQuery, 1)
	case "exercise_week_2":
		b.handleExerciseWeekCallback(callbackQuery, 2)
	case "exercise_week_3":
		b.handleExerciseWeekCallback(callbackQuery, 3)
	case "exercise_week_4":
		b.handleExerciseWeekCallback(callbackQuery, 4)
	default:
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
		"`/setwelcome Добро пожаловать в Lovifyy Bot! 💕 Я помогу укрепить ваши отношения через упражнения и дневник.`\n\n" +
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
				tgbotapi.NewInlineKeyboardButtonData("💬 Обычная беседа", "chat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗓️ Упражнения недели", "advice"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Мини дневник", "diary"),
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
			"`/setwelcome Добро пожаловать в Lovifyy Bot! 💕 Я помогу укрепить ваши отношения через упражнения и дневник.`\n\n" +
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

	// Если состояние пустое (пользователь еще не выбрал режим), показываем главное меню
	if userState == "" {
		response := "Привет! 👋 Выберите режим работы:"

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Обычная беседа", "chat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗓️ Упражнения недели", "advice"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Мини дневник", "diary"),
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
				tgbotapi.NewInlineKeyboardButtonData("💬 Обычная беседа", "chat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗓️ Упражнения недели", "advice"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Мини дневник", "diary"),
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
