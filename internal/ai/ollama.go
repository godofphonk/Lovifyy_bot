package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	return &OllamaClient{
		baseURL: "http://localhost:11434", // Ollama работает локально на порту 11434
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

	// Отправляем запрос к ЛОКАЛЬНОМУ серверу Ollama
	resp, err := http.Post(c.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
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

	return ollamaResp.Response, nil
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
