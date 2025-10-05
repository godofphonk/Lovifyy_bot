package main

import (
	"log"
	"os"

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

	// Получаем API ключ OpenAI
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Fatal("OPENAI_API_KEY не установлен")
	}

	// Создаем и запускаем бота
	bot := NewBot(botToken, openaiAPIKey)
	log.Println("Бот запущен...")
	bot.Start()
}
