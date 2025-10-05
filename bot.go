package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

type Bot struct {
	telegram *tgbotapi.BotAPI
	openai   *openai.Client
}

// NewBot создает новый экземпляр бота
func NewBot(telegramToken, openaiAPIKey string) *Bot {
	// Инициализируем Telegram бота
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Ошибка создания Telegram бота:", err)
	}

	bot.Debug = false
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Инициализируем OpenAI клиента
	openaiClient := openai.NewClient(openaiAPIKey)

	return &Bot{
		telegram: bot,
		openai:   openaiClient,
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
	log.Printf("Получено сообщение от %s: %s", message.From.UserName, message.Text)

	// Обработка команд
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Обработка обычных сообщений через ИИ
	b.handleAIMessage(message)
}

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendMessage(message.Chat.ID, 
			"Привет! 👋 Я ИИ-бот, готовый помочь вам с любыми вопросами.\n\n"+
			"Просто напишите мне что-нибудь, и я отвечу!\n\n"+
			"Доступные команды:\n"+
			"/help - показать справку\n"+
			"/clear - очистить контекст разговора")
	case "help":
		b.sendMessage(message.Chat.ID,
			"🤖 Справка по боту:\n\n"+
			"Я использую искусственный интеллект для ответов на ваши вопросы.\n"+
			"Вы можете спросить меня о чем угодно!\n\n"+
			"Команды:\n"+
			"/start - начать работу с ботом\n"+
			"/help - показать эту справку\n"+
			"/clear - очистить контекст разговора\n\n"+
			"Просто напишите мне сообщение, и я отвечу! 😊")
	case "clear":
		// Здесь можно добавить логику очистки контекста пользователя
		b.sendMessage(message.Chat.ID, "Контекст разговора очищен! 🧹")
	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для получения справки.")
	}
}

// handleAIMessage обрабатывает сообщения через ИИ
func (b *Bot) handleAIMessage(message *tgbotapi.Message) {
	// Отправляем индикатор печати
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Получаем ответ от ИИ
	response, err := b.getAIResponse(message.Text)
	if err != nil {
		log.Printf("Ошибка получения ответа от ИИ: %v", err)
		b.sendMessage(message.Chat.ID, "Извините, произошла ошибка при обработке вашего сообщения. Попробуйте еще раз.")
		return
	}

	// Отправляем ответ пользователю
	b.sendMessage(message.Chat.ID, response)
}

// getAIResponse получает ответ от OpenAI
func (b *Bot) getAIResponse(userMessage string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "Ты полезный ИИ-ассистент в Telegram боте. Отвечай на русском языке, " +
					"будь дружелюбным и полезным. Давай краткие, но информативные ответы.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userMessage,
			},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := b.openai.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса к OpenAI: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от OpenAI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// sendMessage отправляет сообщение пользователю
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	
	_, err := b.telegram.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
