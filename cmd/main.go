package main

import (
	"log"
	"os"

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

	// Создаем и запускаем бота
	telegramBot := bot.NewBot(botToken)
	log.Println("🚀 Lovifyy Bot запущен...")
	log.Println("💾 История сохраняется для каждого пользователя")
	log.Println("🤖 Используется локальная модель: Qwen 3:8B")
	telegramBot.Start()
}
