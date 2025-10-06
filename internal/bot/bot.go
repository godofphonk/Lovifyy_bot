package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"Lovifyy_bot/internal/ai"
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
		return "chat" // по умолчанию режим обычной беседы
	}
	return state
}

// Bot представляет Telegram бота с ИИ
type Bot struct {
	telegram     *tgbotapi.BotAPI
	ai           *ai.OllamaClient
	history      *history.Manager
	rateLimiter  *RateLimiter
	systemPrompt string
	adminIDs     []int64
	userStates   map[int64]string // состояния пользователей (chat, diary)
	stateMutex   sync.RWMutex     // мьютекс для безопасного доступа к состояниям
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
		{Command: "adminhelp", Description: "👑 Админ-панель"},
	}
	
	setCommands := tgbotapi.NewSetMyCommands(commands...)
	if _, err := bot.Request(setCommands); err != nil {
		log.Printf("⚠️ Не удалось установить команды: %v", err)
	} else {
		log.Println("✅ Команды для меню установлены!")
	}

	// Инициализируем AI клиента (используем легкую модель)
	aiClient := ai.NewOllamaClient("gemma3:1b")
	
	// Проверяем доступность AI
	if err := aiClient.TestConnection(); err != nil {
		log.Fatal("AI недоступен:", err)
	}
	log.Println("✅ AI подключен успешно!")

	// Инициализируем систему истории
	historyManager := history.NewManager()
	log.Println("✅ Система истории инициализирована!")

	return &Bot{
		telegram:     bot,
		ai:           aiClient,
		history:      historyManager,
		rateLimiter:  NewRateLimiter(),
		systemPrompt: systemPrompt,
		adminIDs:     adminIDs,
		userStates:   make(map[int64]string),
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
	case "adminhelp":
		b.handleAdminHelpCallback(callbackQuery)
	case "prompt":
		b.handlePromptCallback(callbackQuery)
	case "setprompt_menu":
		b.handleSetPromptMenuCallback(callbackQuery)
	default:
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
	userID := callbackQuery.From.ID
	
	// Получаем всю историю пользователя для анализа
	history, err := b.history.GetUserHistory(userID, 0) // 0 = вся история
	if err != nil || len(history) == 0 {
		b.sendMessage(callbackQuery.Message.Chat.ID, "🗓️ У нас пока нет истории общения для анализа. Сначала поговорите со мной в режиме /chat, а затем я смогу предложить вам персональные упражнения на неделю!")
		return
	}
	
	// Отправляем индикатор печати
	typing := tgbotapi.NewChatAction(callbackQuery.Message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)
	
	// Формируем специальный промпт для упражнений на неделю
	advicePrompt := "Ты коуч по продуктивности и благополучию. Проанализируй всю историю общения с пользователем и составь персональные упражнения на неделю. Учти его цели, интересы и сложности, проявившиеся в беседе. Дай конкретный, практичный недельный план с шагами на каждый день (понедельник-воскресенье).\n\n"
	
	// Добавляем всю историю для анализа
	advicePrompt += "История общения с пользователем:\n"
	for _, msg := range history {
		advicePrompt += fmt.Sprintf("Пользователь: %s\nБот: %s\n\n", msg.Message, msg.Response)
	}
	
	advicePrompt += "На основе всей этой истории общения составь для пользователя персональные упражнения на неделю (краткие задания на каждый день с пояснениями и критериями выполнения):"
	
	// Получаем упражнения от ИИ
	advice, err := b.ai.Generate(advicePrompt)
	if err != nil {
		log.Printf("Ошибка получения упражнений недели: %v", err)
		b.sendMessage(callbackQuery.Message.Chat.ID, "Извините, произошла ошибка при подготовке плана упражнений. Попробуйте позже.")
		return
	}
	
	// Очищаем ответ
	advice = strings.TrimSpace(advice)
	
	// Отправляем упражнения недели
	response := "🗓️ **Ваши упражнения на неделю:**\n\n" + advice
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleDiaryCallback обрабатывает нажатие кнопки "Мини дневник"
func (b *Bot) handleDiaryCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	b.setUserState(userID, "diary")
	
	response := "📝 Режим мини дневника активирован!\n\n" +
		"Теперь просто напишите свои мысли, заметки или события дня. " +
		"Я буду подтверждать, что ваши записи сохранены.\n\n" +
		"Это ваше личное пространство для записей и размышлений."
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleAdminHelpCallback обрабатывает нажатие кнопки "Админ-панель"
func (b *Bot) handleAdminHelpCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	
	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}
	
	response := "👑 **Админ-панель Lovifyy Bot**\n\n" +
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
	
	response := fmt.Sprintf("🤖 **Текущий системный промпт:**\n\n%s\n\n💡 Для изменения используйте:\n/setprompt <новый промпт>", b.systemPrompt)
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleSetPromptMenuCallback обрабатывает нажатие кнопки "Изменить промпт"
func (b *Bot) handleSetPromptMenuCallback(callbackQuery *tgbotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	
	if !b.isAdmin(userID) {
		b.sendMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		return
	}
	
	response := "✏️ **Изменение системного промпта**\n\n" +
		"Отправьте команду в формате:\n" +
		"`/setprompt <новый промпт>`\n\n" +
		"💡 **Готовые варианты:**\n\n" +
		"**Психолог:**\n" +
		"`/setprompt Ты опытный психолог, который помогает людям с их личными проблемами. Будь сочувствующим и давай полезные советы.`\n\n" +
		"**Дружелюбный помощник:**\n" +
		"`/setprompt Ты дружелюбный помощник, готовый ответить на любые вопросы. Будь позитивным и полезным.`\n\n" +
		"**Программист:**\n" +
		"`/setprompt Ты программист-эксперт, специализирующийся на Go и веб-разработке. Помогай с кодом и объясняй концепции.`"
	b.sendMessage(callbackQuery.Message.Chat.ID, response)
}

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	
	switch message.Command() {
	case "start":
		response := "Привет! 👋 Я Lovifyy Bot - ваш персональный помощник!\n\n" +
			"🤖 Работаю полностью локально с ИИ\n" +
			"💾 Запоминаю всю нашу беседу\n" +
			"🗓️ Готов подготовить упражнения на неделю на основе нашего общения\n\n" +
			"Выберите режим работы:"
		
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
		// Получаем всю историю пользователя для анализа
		history, err := b.history.GetUserHistory(userID, 0) // 0 = вся история
		if err != nil || len(history) == 0 {
			b.sendMessage(message.Chat.ID, "🗓️ У нас пока нет истории общения для анализа. Сначала поговорите со мной в режиме /chat, а затем я смогу предложить вам персональные упражнения на неделю!")
			return
		}
		
		// Отправляем индикатор печати
		typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
		b.telegram.Send(typing)
		
		// Формируем специальный промпт для упражнений на неделю
		advicePrompt := "Ты коуч по продуктивности и благополучию. Проанализируй всю историю общения с пользователем и составь персональные упражнения на неделю. Учти его цели, интересы и сложности, проявившиеся в беседе. Дай конкретный, практичный недельный план с шагами на каждый день (понедельник-воскресенье).\n\n"
		
		// Добавляем всю историю для анализа
		advicePrompt += "История общения с пользователем:\n"
		for _, msg := range history {
			advicePrompt += fmt.Sprintf("Пользователь: %s\nБот: %s\n\n", msg.Message, msg.Response)
		}
		
		advicePrompt += "На основе всей этой истории общения составь для пользователя персональные упражнения на неделю (краткие задания на каждый день с пояснениями и критериями выполнения):"
		
		// Получаем упражнения от ИИ
		advice, err := b.ai.Generate(advicePrompt)
		if err != nil {
			log.Printf("Ошибка получения упражнений недели: %v", err)
			b.sendMessage(message.Chat.ID, "Извините, произошла ошибка при подготовке плана упражнений. Попробуйте позже.")
			return
		}
		
		// Очищаем ответ
		advice = strings.TrimSpace(advice)
		
		// Отправляем упражнения недели
		response := "🗓️ **Ваши упражнения на неделю:**\n\n" + advice
		b.sendMessage(message.Chat.ID, response)
		
	case "diary":
		b.setUserState(userID, "diary")
		response := "📝 Режим мини дневника активирован!\n\n" +
			"Теперь просто напишите свои мысли, заметки или события дня. " +
			"Я буду подтверждать, что ваши записи сохранены.\n\n" +
			"Это ваше личное пространство для записей и размышлений."
		b.sendMessage(message.Chat.ID, response)
		
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
		
	case "adminhelp":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}
		
		response := "👑 **Админ-панель Lovifyy Bot**\n\n" +
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
		
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ReplyMarkup = adminKeyboard
		b.telegram.Send(msg)
		
	case "prompt":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}
		
		response := fmt.Sprintf("🤖 **Текущий системный промпт:**\n\n%s\n\n💡 Для изменения используйте:\n/setprompt <новый промпт>", b.systemPrompt)
		b.sendMessage(message.Chat.ID, response)
		
	case "setprompt_menu":
		if !b.isAdmin(userID) {
			b.sendMessage(message.Chat.ID, "❌ Эта команда доступна только администраторам.")
			return
		}
		
		response := "✏️ **Изменение системного промпта**\n\n" +
			"Отправьте команду в формате:\n" +
			"`/setprompt <новый промпт>`\n\n" +
			"💡 **Готовые варианты:**\n\n" +
			"**Психолог:**\n" +
			"`/setprompt Ты опытный психолог, который помогает людям с их личными проблемами. Будь сочувствующим и давай полезные советы.`\n\n" +
			"**Дружелюбный помощник:**\n" +
			"`/setprompt Ты дружелюбный помощник, готовый ответить на любые вопросы. Будь позитивным и полезным.`\n\n" +
			"**Программист:**\n" +
			"`/setprompt Ты программист-эксперт, специализирующийся на Go и веб-разработке. Помогай с кодом и объясняй концепции.`"
		b.sendMessage(message.Chat.ID, response)
		
	case "help":
		response := "🤖 **Справка по Lovifyy Bot:**\n\n" +
			"💬 **/chat** - режим обычной беседы\n" +
			"🗓️ **/advice** - упражнения недели\n" +
			"📝 **/diary** - мини дневник\n" +
			"🚀 **/start** - главное меню\n\n" +
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
	
	// Если пользователь в режиме дневника, просто подтверждаем запись
	if userState == "diary" {
		// Сохраняем запись в историю как дневниковую запись
		diaryResponse := "📝 Записано"
		err := b.history.SaveMessage(userID, username, message.Text, diaryResponse, "diary")
		if err != nil {
			log.Printf("Ошибка сохранения записи дневника: %v", err)
		}
		
		// Отправляем подтверждение
		b.sendMessage(message.Chat.ID, diaryResponse)
		return
	}

	// Отправляем индикатор печати для обычного режима
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Получаем контекст из истории (последние 5 сообщений)
	context := b.history.GetRecentContext(userID, 5)
	
	// Формируем промпт с системным промптом и контекстом
	prompt := b.systemPrompt + "\n\n"
	if context != "" {
		prompt += context + "\n"
	}
	prompt += fmt.Sprintf("Пользователь: %s\nБот:", message.Text)

	// Получаем ответ от ИИ
	response, err := b.ai.Generate(prompt)
	if err != nil {
		log.Printf("Ошибка получения ответа от ИИ: %v", err)
		b.sendMessage(message.Chat.ID, "Извините, произошла ошибка при обработке вашего сообщения. Попробуйте еще раз.")
		return
	}

	// Очищаем ответ
	response = strings.TrimSpace(response)
	
	// Сохраняем в историю
	err = b.history.SaveMessage(userID, username, message.Text, response, "gemma3:1b")
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
