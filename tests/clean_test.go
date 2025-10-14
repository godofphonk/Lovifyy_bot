package tests

import (
	"fmt"
	"os"
	"testing"
	"github.com/godofphonk/lovifyy-bot/internal/ai"
)

func TestOpenAIClient(t *testing.T) {
	fmt.Println("🧪 Тестируем OpenAI клиент...")
	
	// Проверяем наличие API ключа
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY не установлен, пропускаем тест")
		return
	}
	
	client := ai.NewOpenAIClient("gpt-4o-mini")
	
	// Проверяем подключение
	if err := client.TestConnection(); err != nil {
		t.Logf("❌ Ошибка подключения к OpenAI: %v", err)
		t.Skip("OpenAI недоступен, пропускаем тест")
		return
	}
	
	// Тестируем простой вопрос
	fmt.Println("\n🤖 Тестируем простой вопрос...")
	response, err := client.Generate("Скажи просто 'Привет!' без лишних слов")
	if err != nil {
		t.Errorf("❌ Ошибка генерации: %v", err)
		return
	}
	
	fmt.Printf("✅ Ответ OpenAI: '%s'\n", response)
	
	// Проверяем что ответ не пустой
	if response == "" {
		t.Error("❌ Получен пустой ответ")
	} else {
		fmt.Println("✅ OpenAI клиент работает корректно!")
	}
}
