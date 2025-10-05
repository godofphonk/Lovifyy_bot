package tests

import (
	"fmt"
	"log"
	"testing"

	"Lovifyy_bot/internal/ai"
)

func TestOllamaConnection(t *testing.T) {
	fmt.Println("🧪 Тестируем локальный Ollama...")
	
	// Создаем клиент для модели Qwen
	client := ai.NewOllamaClient("qwen3:8b")
	
	// Проверяем доступность
	fmt.Println("🔍 Проверяем подключение к Ollama...")
	if err := client.TestConnection(); err != nil {
		t.Errorf("❌ Ошибка подключения к Ollama: %v", err)
		return
	}
	
	// Тестируем генерацию
	fmt.Println("\n🤖 Тестируем генерацию ответов...")
	
	testQuestions := []string{
		"Привет! Как дела?",
		"Расскажи анекдот",
		"Что ты умеешь?",
	}
	
	for i, question := range testQuestions {
		fmt.Printf("\n%d. Вопрос: %s\n", i+1, question)
		
		response, err := client.Generate(question)
		if err != nil {
			t.Errorf("❌ Ошибка генерации для вопроса '%s': %v", question, err)
			continue
		}
		
		if response == "" {
			t.Errorf("❌ Пустой ответ для вопроса '%s'", question)
			continue
		}
		
		fmt.Printf("   🤖 Ответ: %s\n", response)
	}
	
	fmt.Println("\n✅ Тест завершен успешно!")
}

func TestOllamaManual() {
	fmt.Println("🧪 Ручной тест Ollama...")
	
	client := ai.NewOllamaClient("qwen3:8b")
	
	if err := client.TestConnection(); err != nil {
		log.Printf("❌ Ошибка: %v\n", err)
		fmt.Println("\n📋 Инструкция по установке:")
		fmt.Println("1. Скачайте Ollama: https://ollama.com/download/windows")
		fmt.Println("2. Установите и запустите")
		fmt.Println("3. Выполните: ollama pull qwen3:8b")
		fmt.Println("4. Запустите этот тест снова")
		return
	}
	
	testQuestions := []string{
		"Привет! Как дела?",
		"Расскажи анекдот",
		"Что ты умеешь?",
	}
	
	fmt.Println("\n🤖 Тестируем генерацию ответов...")
	for i, question := range testQuestions {
		fmt.Printf("\n%d. Вопрос: %s\n", i+1, question)
		
		response, err := client.Generate(question)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
			continue
		}
		
		fmt.Printf("   🤖 Ответ: %s\n", response)
	}
	
	fmt.Println("\n✅ Тест завершен!")
	fmt.Println("💡 Если все работает - можно запускать бота!")
}
