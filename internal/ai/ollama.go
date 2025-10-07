package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// OllamaClient клиент для работы с локальным Ollama
type OllamaClient struct {
	baseURL string
	model   string
}

// OllamaRequest структура запроса к Ollama
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse структура ответа от Ollama
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewOllamaClient создает новый клиент Ollama
func NewOllamaClient(model string) *OllamaClient {
	// Получаем URL Ollama из переменной окружения (для Docker)
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Дефолтный URL для локальной разработки
	}
	
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
	}
}

// Generate генерирует ответ через локальную модель
func (c *OllamaClient) Generate(prompt string) (string, error) {
	// Создаем запрос
	reqData := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("ошибка создания JSON: %w", err)
	}

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 120 * time.Second, // Таймаут 120 секунд для генерации
	}
	
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	
	// Создаем запрос с контекстом
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Отправляем запрос к ЛОКАЛЬНОМУ серверу Ollama
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка подключения к Ollama (убедитесь что Ollama запущен): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка Ollama: статус %d", resp.StatusCode)
	}

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	// Очищаем ответ от блоков размышлений
	response := c.cleanResponse(ollamaResp.Response)
	return response, nil
}

// IsAvailable проверяет доступность Ollama
func (c *OllamaClient) IsAvailable() bool {
	resp, err := http.Get(c.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// TestConnection тестирует подключение к Ollama
func (c *OllamaClient) TestConnection() error {
	if !c.IsAvailable() {
		return fmt.Errorf("Ollama недоступен. Убедитесь что:\n1. Ollama установлен\n2. Ollama запущен\n3. Модель %s загружена", c.model)
	}

	// Тестовый запрос
	response, err := c.Generate("Скажи просто 'Работаю!'")
	if err != nil {
		return fmt.Errorf("ошибка тестового запроса: %w", err)
	}

	fmt.Printf("✅ Ollama работает! Ответ: %s\n", response)
	return nil
}

// cleanResponse очищает ответ от блоков размышлений и лишнего текста
func (c *OllamaClient) cleanResponse(response string) string {
	// Удаляем блоки <think>...</think> (включая многострочные)
	thinkRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")
	
	// Удаляем блоки </think> без открывающего тега (на случай ошибок парсинга)
	thinkEndRegex := regexp.MustCompile(`(?s).*?</think>`)
	cleaned = thinkEndRegex.ReplaceAllString(cleaned, "")
	
	// Удаляем строки, содержащие только </think>
	thinkLineRegex := regexp.MustCompile(`(?m)^.*</think>.*$\n?`)
	cleaned = thinkLineRegex.ReplaceAllString(cleaned, "")
	
	// Удаляем лишние пробелы и переносы строк в начале и конце
	cleaned = strings.TrimSpace(cleaned)
	
	// Удаляем множественные пустые строки
	multipleNewlines := regexp.MustCompile(`\n\s*\n\s*\n`)
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")
	
	// Если после очистки остался пустой ответ, возвращаем дефолтное сообщение
	if cleaned == "" {
		cleaned = "Извините, не могу сформулировать ответ."
	}
	
	return cleaned
}
