package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
	"golang.org/x/net/proxy"
)

// OpenAIClient клиент для работы с OpenAI API
type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
}

// OpenAIMessage представляет сообщение в формате OpenAI
type OpenAIMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// OpenAIRequest структура запроса к OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// OpenAIResponse структура ответа от OpenAI API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIClient создает новый клиент OpenAI
func NewOpenAIClient(model string) *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY не установлен в переменных окружения")
	}

	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		model:   model,
	}
}

// Generate генерирует ответ через OpenAI API с историей сообщений
func (c *OpenAIClient) GenerateWithHistory(messages []OpenAIMessage) (string, error) {
	// Создаем запрос
	reqData := OpenAIRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   1500,
		Temperature: 0.7,
		Stream:      false,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("ошибка создания JSON: %w", err)
	}

	// Создаем HTTP клиент с поддержкой прокси
	client := c.createHTTPClient()

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем запрос с контекстом
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка подключения к OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ошибка OpenAI API: статус %d, ответ: %s", resp.StatusCode, string(body))
	}

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от OpenAI")
	}

	return openaiResp.Choices[0].Message.Content, nil
}

// Generate простая генерация для совместимости с существующим кодом
func (c *OpenAIClient) Generate(prompt string) (string, error) {
	messages := []OpenAIMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}
	return c.GenerateWithHistory(messages)
}

// TestConnection тестирует подключение к OpenAI API
func (c *OpenAIClient) TestConnection() error {
	// Простой тестовый запрос
	response, err := c.Generate("Скажи просто 'Работаю!'")
	if err != nil {
		return fmt.Errorf("ошибка тестового запроса: %w", err)
	}

	fmt.Printf("✅ OpenAI API работает! Ответ: %s\n", response)
	return nil
}

// SetModel изменяет модель
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}

// GetModel возвращает текущую модель
func (c *OpenAIClient) GetModel() string {
	return c.model
}

// createHTTPClient создает HTTP клиент с поддержкой прокси
func (c *OpenAIClient) createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Проверяем переменные окружения для прокси
	proxyURL := os.Getenv("HTTPS_PROXY")
	if proxyURL == "" {
		proxyURL = os.Getenv("HTTP_PROXY")
	}
	if proxyURL == "" {
		proxyURL = os.Getenv("ALL_PROXY")
	}

	if proxyURL != "" {
		fmt.Printf("🌐 Используем прокси: %s\n", proxyURL)
		
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			fmt.Printf("❌ Ошибка парсинга прокси URL: %v\n", err)
			return client
		}

		// Поддержка SOCKS5 прокси
		if parsedURL.Scheme == "socks5" {
			dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, nil, proxy.Direct)
			if err != nil {
				fmt.Printf("❌ Ошибка создания SOCKS5 прокси: %v\n", err)
				return client
			}
			
			client.Transport = &http.Transport{
				Dial: dialer.Dial,
			}
		} else {
			// HTTP/HTTPS прокси
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsedURL),
			}
		}
	}

	return client
}
