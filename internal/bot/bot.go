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

// Bot представляет Telegram бота с ИИ
type Bot struct {
	telegram     *tgbotapi.BotAPI
	ai           *ai.OllamaClient
	history      *history.Manager
	rateLimiter  *RateLimiter
	systemPrompt string
	adminIDs     []int64
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
		{Command: "help", Description: "❓ Показать справку"},
		{Command: "clear", Description: "🧹 Очистить историю разговора"},
		{Command: "stats", Description: "📊 Показать статистику"},
		{Command: "prompt", Description: "🤖 Показать системный промпт"},
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
	
	// Подтверждаем получение callback
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	b.telegram.Request(callback)
	
	// Создаем фейковое сообщение для использования существующих обработчиков команд
	fakeMessage := &tgbotapi.Message{
		MessageID: callbackQuery.Message.MessageID,
		From:      callbackQuery.From,
		Chat:      callbackQuery.Message.Chat,
		Date:      callbackQuery.Message.Date,
		Text:      "/" + data, // Превращаем callback data в команду
	}
	
	// Обрабатываем как команду
	b.handleCommand(fakeMessage)
}

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	
	switch message.Command() {
	case "start":
		response := "Привет! 👋 Я Lovifyy Bot с локальным ИИ!\n\n" +
			"🤖 Работаю полностью локально - без лимитов и платежей\n" +
			"💾 Сохраняю историю наших разговоров\n" +
			"🚀 Готов отвечать на любые вопросы!"
		
		// Создаем inline клавиатуру с основными командами
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❓ Справка", "help"),
				tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🧹 Очистить историю", "clear"),
				tgbotapi.NewInlineKeyboardButtonData("🤖 Промпт", "prompt"),
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
		
	case "help":
		response := "🤖 Справка по Lovifyy Bot:\n\n" +
			"Я использую локальный ИИ Qwen для ответов на ваши вопросы.\n" +
			"Все работает на сервере без внешних API!\n\n" +
			"Команды:\n" +
			"/start - начать работу\n" +
			"/help - эта справка\n" +
			"/clear - очистить историю разговора\n" +
			"/stats - показать статистику\n" +
			"/prompt - показать системный промпт\n\n" +
			"Просто напишите мне любое сообщение! 😊"
		b.sendMessage(message.Chat.ID, response)
		
	case "clear":
		err := b.history.ClearUserHistory(userID)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Ошибка при очистке истории.")
			return
		}
		response := "🧹 История разговора очищена!\n\nТеперь я не помню наши предыдущие сообщения."
		b.sendMessage(message.Chat.ID, response)
		
	case "stats":
		count, firstMsg, err := b.history.GetStats(userID)
		if err != nil || count == 0 {
			b.sendMessage(message.Chat.ID, "📊 У вас пока нет истории сообщений.")
			return
		}
		
		response := fmt.Sprintf("📊 Ваша статистика:\n\n"+
			"💬 Всего сообщений: %d\n"+
			"📅 Первое сообщение: %s\n"+
			"🤖 Модель: Qwen 3:8B (локальная)\n"+
			"💾 История сохраняется локально",
			count, firstMsg.Format("02.01.2006 15:04"))
		b.sendMessage(message.Chat.ID, response)
		
	case "prompt":
		response := fmt.Sprintf("🤖 Текущий системный промпт:\n\n%s", b.systemPrompt)
		if b.isAdmin(userID) {
			response += "\n\n💡 Для изменения используйте: /setprompt <новый промпт>"
		} else {
			response += "\n\n💡 Для изменения промпта обратитесь к администратору."
		}
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
		
		response := "👑 Справка для администраторов:\n\n" +
			"🔧 Доступные команды:\n" +
			"/setprompt <текст> - изменить системный промпт бота\n" +
			"/prompt - посмотреть текущий промпт\n" +
			"/adminhelp - эта справка\n\n" +
			"💡 Примеры промптов:\n" +
			"• Ты дружелюбный помощник\n" +
			"• Ты опытный психолог\n" +
			"• Ты программист-эксперт\n\n" +
			"⚠️ Изменения применяются сразу для всех пользователей!"
		b.sendMessage(message.Chat.ID, response)
		
	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для получения справки.")
	}
}

// handleAIMessage обрабатывает сообщения через ИИ с учетом истории
func (b *Bot) handleAIMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}

	// Отправляем индикатор печати
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
	err = b.history.SaveMessage(userID, username, message.Text, response, "qwen3:8b")
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
