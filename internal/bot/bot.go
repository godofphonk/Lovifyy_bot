package bot

import (
	"fmt"
	"log"
	"strings"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/history"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота с ИИ
type Bot struct {
	telegram *tgbotapi.BotAPI
	ai       *ai.OllamaClient
	history  *history.Manager
}

// NewBot создает новый экземпляр бота
func NewBot(telegramToken string) *Bot {
	// Инициализируем Telegram бота
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Ошибка создания Telegram бота:", err)
	}

	bot.Debug = false
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Инициализируем AI клиента
	aiClient := ai.NewOllamaClient("qwen3:8b")
	
	// Проверяем доступность AI
	if err := aiClient.TestConnection(); err != nil {
		log.Fatal("AI недоступен:", err)
	}
	log.Println("✅ AI подключен успешно!")

	// Инициализируем систему истории
	historyManager := history.NewManager()
	log.Println("✅ Система истории инициализирована!")

	return &Bot{
		telegram: bot,
		ai:       aiClient,
		history:  historyManager,
	}
}

// Start запускает бота
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.telegram.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go b.handleMessage(update.Message)
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

	// Обработка команд
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Обработка обычных сообщений через ИИ с историей
	b.handleAIMessage(message)
}

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	
	switch message.Command() {
	case "start":
		response := "Привет! 👋 Я Lovifyy Bot с локальным ИИ!\n\n" +
			"🤖 Работаю полностью локально - без лимитов и платежей\n" +
			"💾 Сохраняю историю наших разговоров\n" +
			"🚀 Готов отвечать на любые вопросы!\n\n" +
			"Доступные команды:\n" +
			"/help - показать справку\n" +
			"/clear - очистить историю\n" +
			"/stats - статистика общения"
		b.sendMessage(message.Chat.ID, response)
		
	case "help":
		response := "🤖 Справка по Lovifyy Bot:\n\n" +
			"Я использую локальный ИИ Qwen для ответов на ваши вопросы.\n" +
			"Все работает на сервере без внешних API!\n\n" +
			"Команды:\n" +
			"/start - начать работу\n" +
			"/help - эта справка\n" +
			"/clear - очистить историю разговора\n" +
			"/stats - показать статистику\n\n" +
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
	
	// Формируем промпт с контекстом
	prompt := "Ты полезный ИИ-ассистент по имени Lovifyy Bot. Отвечай на русском языке, будь дружелюбным и полезным.\n\n"
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
