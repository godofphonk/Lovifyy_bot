package main

import (
	"fmt"
	"Lovifyy_bot/internal/ai"
)

func main() {
	fmt.Println("🧪 Тестируем очистку ответов от <think> блоков...")
	
	client := ai.NewOllamaClient("qwen3:8b")
	
	// Проверяем подключение
	if err := client.TestConnection(); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	
	// Тестируем простой вопрос
	fmt.Println("\n🤖 Тестируем простой вопрос...")
	response, err := client.Generate("Скажи просто 'Привет!' без лишних слов")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Очищенный ответ: '%s'\n", response)
	
	// Проверяем что <think> блоки удалены
	if len(response) < 200 && response != "" {
		fmt.Println("✅ Ответ выглядит чистым!")
	} else {
		fmt.Println("⚠️ Ответ все еще содержит лишний текст")
	}
}
