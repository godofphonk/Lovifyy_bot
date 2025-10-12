package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UserInfo содержит информацию о пользователе
type UserInfo struct {
	UserID   int64     `json:"user_id"`
	Username string    `json:"username"`
	IsActive bool      `json:"is_active"`
	JoinedAt time.Time `json:"joined_at"`
	LastSeen time.Time `json:"last_seen"`
}

// UserStorage управляет хранением пользователей в JSON файле
type UserStorage struct {
	filePath string
	mutex    sync.RWMutex
}

// NewUserStorage создает новое хранилище пользователей
func NewUserStorage(dataDir string) *UserStorage {
	os.MkdirAll(dataDir, 0755)
	return &UserStorage{
		filePath: filepath.Join(dataDir, "users.json"),
	}
}

// AddUser добавляет или обновляет пользователя
func (us *UserStorage) AddUser(userID int64, username string) error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	users, err := us.loadUsers()
	if err != nil {
		users = make(map[int64]*UserInfo)
	}

	// Проверяем, существует ли пользователь
	if existingUser, exists := users[userID]; exists {
		// Обновляем информацию
		existingUser.Username = username
		existingUser.LastSeen = time.Now()
		existingUser.IsActive = true
	} else {
		// Создаем нового пользователя
		users[userID] = &UserInfo{
			UserID:   userID,
			Username: username,
			IsActive: true,
			JoinedAt: time.Now(),
			LastSeen: time.Now(),
		}
	}

	return us.saveUsers(users)
}

// GetAllActiveUsers возвращает всех активных пользователей
func (us *UserStorage) GetAllActiveUsers() ([]UserInfo, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	users, err := us.loadUsers()
	if err != nil {
		return nil, err
	}

	var activeUsers []UserInfo
	for _, user := range users {
		if user.IsActive {
			activeUsers = append(activeUsers, *user)
		}
	}

	return activeUsers, nil
}

// GetUserIDs возвращает список ID всех активных пользователей
func (us *UserStorage) GetUserIDs() ([]int64, error) {
	users, err := us.GetAllActiveUsers()
	if err != nil {
		return nil, err
	}

	var userIDs []int64
	for _, user := range users {
		userIDs = append(userIDs, user.UserID)
	}

	return userIDs, nil
}

// UpdateLastSeen обновляет время последней активности пользователя
func (us *UserStorage) UpdateLastSeen(userID int64) error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	users, err := us.loadUsers()
	if err != nil {
		return err
	}

	if user, exists := users[userID]; exists {
		user.LastSeen = time.Now()
		return us.saveUsers(users)
	}

	return nil
}

// DeactivateUser деактивирует пользователя
func (us *UserStorage) DeactivateUser(userID int64) error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	users, err := us.loadUsers()
	if err != nil {
		return err
	}

	if user, exists := users[userID]; exists {
		user.IsActive = false
		return us.saveUsers(users)
	}

	return nil
}

// loadUsers загружает пользователей из JSON файла
func (us *UserStorage) loadUsers() (map[int64]*UserInfo, error) {
	data, err := os.ReadFile(us.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int64]*UserInfo), nil
		}
		return nil, err
	}

	var users map[int64]*UserInfo
	err = json.Unmarshal(data, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// saveUsers сохраняет пользователей в JSON файл
func (us *UserStorage) saveUsers(users map[int64]*UserInfo) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(us.filePath, data, 0644)
}

// GetUserCount возвращает количество активных пользователей
func (us *UserStorage) GetUserCount() (int, error) {
	users, err := us.GetAllActiveUsers()
	if err != nil {
		return 0, err
	}
	return len(users), nil
}
