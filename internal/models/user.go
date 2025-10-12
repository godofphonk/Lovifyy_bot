package models

import (
	"sync"
	"time"
)

// UserState представляет состояние пользователя в боте.
// Содержит информацию о текущем состоянии пользователя и дополнительные данные.
type UserState struct {
	State     string    `json:"state"`
	Data      string    `json:"data,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User представляет пользователя бота
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// RateLimitEntry представляет запись для rate limiting
type RateLimitEntry struct {
	LastMessage time.Time `json:"last_message"`
	Count       int       `json:"count"`
}

// UserManager управляет состояниями и данными пользователей
type UserManager struct {
	states      map[int64]*UserState
	rateLimits  map[int64]*RateLimitEntry
	adminIDs    []int64
	mutex       sync.RWMutex
}

// NewUserManager создает новый менеджер пользователей
func NewUserManager(adminIDs []int64) *UserManager {
	return &UserManager{
		states:     make(map[int64]*UserState),
		rateLimits: make(map[int64]*RateLimitEntry),
		adminIDs:   adminIDs,
	}
}

// SetState устанавливает состояние пользователя
func (um *UserManager) SetState(userID int64, state string) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	um.states[userID] = &UserState{
		State:     state,
		UpdatedAt: time.Now(),
	}
}

// GetState получает состояние пользователя
func (um *UserManager) GetState(userID int64) string {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	if state, exists := um.states[userID]; exists {
		return state.State
	}
	return ""
}

// SetStateData устанавливает данные состояния пользователя
func (um *UserManager) SetStateData(userID int64, state, data string) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	um.states[userID] = &UserState{
		State:     state,
		Data:      data,
		UpdatedAt: time.Now(),
	}
}

// GetStateData получает данные состояния пользователя
func (um *UserManager) GetStateData(userID int64) (string, string) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	if state, exists := um.states[userID]; exists {
		return state.State, state.Data
	}
	return "", ""
}

// IsAdmin проверяет, является ли пользователь администратором
func (um *UserManager) IsAdmin(userID int64) bool {
	for _, adminID := range um.adminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// IsRateLimited проверяет rate limiting для пользователя
func (um *UserManager) IsRateLimited(userID int64, limit time.Duration) bool {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	now := time.Now()
	entry, exists := um.rateLimits[userID]
	
	if !exists {
		um.rateLimits[userID] = &RateLimitEntry{
			LastMessage: now,
			Count:       1,
		}
		return false
	}
	
	if now.Sub(entry.LastMessage) < limit {
		entry.Count++
		return entry.Count > 1 // Разрешаем только 1 сообщение в период
	}
	
	// Сбрасываем счетчик
	entry.LastMessage = now
	entry.Count = 1
	return false
}

// ClearState очищает состояние пользователя
func (um *UserManager) ClearState(userID int64) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	delete(um.states, userID)
}

// GetAdminIDs возвращает список ID администраторов
func (um *UserManager) GetAdminIDs() []int64 {
	return um.adminIDs
}
