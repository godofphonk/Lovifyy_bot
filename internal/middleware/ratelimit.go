package middleware

import (
	"time"

	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RateLimitMiddleware представляет middleware для ограничения частоты запросов
type RateLimitMiddleware struct {
	userManager *models.UserManager
	limit       time.Duration
}

// NewRateLimitMiddleware создает новый middleware для rate limiting
func NewRateLimitMiddleware(userManager *models.UserManager, limit time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		userManager: userManager,
		limit:       limit,
	}
}

// Handler представляет обработчик сообщений
type Handler func(update tgbotapi.Update) error

// Apply применяет rate limiting к обработчику
func (rl *RateLimitMiddleware) Apply(handler Handler) Handler {
	return func(update tgbotapi.Update) error {
		var userID int64
		
		// Получаем ID пользователя из разных типов обновлений
		if update.Message != nil {
			userID = update.Message.From.ID
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else {
			// Если не можем определить пользователя, пропускаем
			return handler(update)
		}
		
		// Проверяем rate limiting
		if rl.userManager.IsRateLimited(userID, rl.limit) {
			// Пользователь превысил лимит, игнорируем запрос
			return nil
		}
		
		// Передаем управление следующему обработчику
		return handler(update)
	}
}

// AdminBypass создает middleware с обходом для администраторов
func (rl *RateLimitMiddleware) AdminBypass(handler Handler) Handler {
	return func(update tgbotapi.Update) error {
		var userID int64
		
		// Получаем ID пользователя
		if update.Message != nil {
			userID = update.Message.From.ID
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else {
			return handler(update)
		}
		
		// Администраторы обходят rate limiting
		if rl.userManager.IsAdmin(userID) {
			return handler(update)
		}
		
		// Для обычных пользователей применяем rate limiting
		if rl.userManager.IsRateLimited(userID, rl.limit) {
			return nil
		}
		
		return handler(update)
	}
}
