package ai

// AIClient общий интерфейс для всех AI клиентов
type AIClient interface {
	Generate(prompt string) (string, error)
	TestConnection() error
}

// HistoryAIClient интерфейс для AI клиентов с поддержкой истории
type HistoryAIClient interface {
	AIClient
	GenerateWithHistory(messages []OpenAIMessage) (string, error)
	SetModel(model string)
	GetModel() string
}
