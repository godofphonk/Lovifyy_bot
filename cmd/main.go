package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"Lovifyy_bot/internal/bot"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные окружения
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден, используем системные переменные окружения")
	}

	// Получаем токен бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	// Получаем системный промпт
	systemPrompt := os.Getenv("SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = "Ты - профессиональный консультант по отношениям и семейный психолог с многолетним опытом работы. Твоя задача - помогать парам и людям в отношениях, давать мудрые советы, поддерживать и направлять к здоровым отношениям. Отвечай на русском языке, будь эмпатичным, понимающим и профессиональным. Твои ответы должны быть практичными и основанными на психологических принципах. ВАЖНО: Ты помнишь всю историю нашего разговора и можешь ссылаться на предыдущие сообщения пользователя."
		log.Println("⚠️ SYSTEM_PROMPT не установлен, используется промпт по умолчанию")
	}

	// Получаем список админов
	adminIDsStr := os.Getenv("ADMIN_IDS")
	var adminIDs []int64
	if adminIDsStr != "" {
		adminIDsList := strings.Split(adminIDsStr, ",")
		for _, idStr := range adminIDsList {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
	}
	log.Printf("👑 Загружено %d администраторов", len(adminIDs))

	// Создаем и запускаем бота
	telegramBot := bot.NewBot(botToken, systemPrompt, adminIDs)
	log.Println("🚀 Lovifyy Bot запущен...")
	log.Println("💾 История сохраняется для каждого пользователя")
	log.Println("🤖 Используется OpenAI модель: GPT-4o-mini")
	telegramBot.Start()
}
