package history

import (
	"os"
	"time"
)

// ChatMessage представляет одно сообщение в истории
type ChatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	Model     string    `json:"model"`
}

// DiaryEntry представляет одну запись в дневнике
type DiaryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Entry     string    `json:"entry"`
	Week      int       `json:"week"`               // номер недели (1-4)
	Type      string    `json:"type"`               // тип записи: questions, joint, personal
	Mood      string    `json:"mood,omitempty"`     // настроение (опционально)
	Tags      []string  `json:"tags,omitempty"`     // теги (опционально)
}

// Manager управляет историей переписки и дневниками
type Manager struct {
	historyDir string
	diaryDir   string
}

// NewManager создает новый менеджер истории
func NewManager() *Manager {
	historyDir := "data/chats"
	diaryDir := "data/diaries"
	os.MkdirAll(historyDir, 0755)
	os.MkdirAll(diaryDir, 0755)
	return &Manager{
		historyDir: historyDir,
		diaryDir:   diaryDir,
	}
}
