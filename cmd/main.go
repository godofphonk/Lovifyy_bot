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
		systemPrompt = "Ты полезный ИИ-ассистент по имени Lovifyy Bot. Отвечай на русском языке, будь дружелюбным и полезным."
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
	log.Println("🤖 Используется локальная модель: Qwen 3:8B")
	telegramBot.Start()
}
