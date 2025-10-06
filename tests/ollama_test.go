package tests

import (
	"fmt"
	"testing"

	"Lovifyy_bot/internal/ai"
)

func TestOllamaConnection(t *testing.T) {
	fmt.Println("🧪 Тестируем локальный Ollama...")
	
	// Создаем клиент для легкой модели
	client := ai.NewOllamaClient("gemma3:1b")
	
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

func TestOllamaManual(t *testing.T) {
	fmt.Println("🧪 Ручной тест Ollama...")
	
	client := ai.NewOllamaClient("gemma3:1b")
	
	if err := client.TestConnection(); err != nil {
		t.Logf("❌ Ошибка: %v\n", err)
		t.Log("\n📋 Инструкция по установке:")
		t.Log("1. Скачайте Ollama: https://ollama.com/download/windows")
		t.Log("2. Установите и запустите")
		t.Log("3. Выполните: ollama pull gemma3:1b")
		t.Log("4. Запустите этот тест снова")
		t.Skip("Ollama недоступен")
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
			t.Logf("   ❌ Ошибка: %v\n", err)
			continue
		}
		
		t.Logf("   🤖 Ответ: %s\n", response)
	}
	
	fmt.Println("\n✅ Тест завершен!")
	fmt.Println("💡 Если все работает - можно запускать бота!")
}
