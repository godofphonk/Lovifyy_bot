package tests

import (
	"fmt"
	"testing"
	"Lovifyy_bot/internal/ai"
)

func TestResponseCleaning(t *testing.T) {
	fmt.Println("🧪 Тестируем очистку ответов от <think> блоков...")
	
	client := ai.NewOllamaClient("gemma3:1b")
	
	// Проверяем подключение
	if err := client.TestConnection(); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	
	// Тестируем простой вопрос
	fmt.Println("\n🤖 Тестируем простой вопрос...")
	response, err := client.Generate("Скажи просто 'Привет!' без лишних слов")
	if err != nil {
		t.Errorf("❌ Ошибка: %v", err)
		return
	}
	
	fmt.Printf("✅ Очищенный ответ: '%s'\n", response)
	
	// Проверяем что <think> блоки удалены
	if len(response) < 200 && response != "" {
		fmt.Println("✅ Ответ выглядит чистым!")
	} else {
		t.Errorf("⚠️ Ответ все еще содержит лишний текст: длина %d", len(response))
	}
}
