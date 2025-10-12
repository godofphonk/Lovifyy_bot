package chat

import (
	"Lovifyy_bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler обрабатывает функциональность чата
type Handler struct {
	bot         *tgbotapi.BotAPI
	userManager *models.UserManager
}

// NewHandler создает новый обработчик чата
func NewHandler(bot *tgbotapi.BotAPI, userManager *models.UserManager) *Handler {
	return &Handler{
		bot:         bot,
		userManager: userManager,
	}
}

// HandleChat обрабатывает нажатие кнопки "Задать вопрос о отношениях"
func (h *Handler) HandleChat(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID
	h.userManager.SetState(userID, "chat")

	response := "💬 Режим обычной беседы активирован!\n\n" +
		"Теперь просто напишите мне любое сообщение, и я отвечу как обычный собеседник. " +
		"Я буду помнить нашу беседу и отвечать в контексте нашего разговора.\n\n" +
		"Чтобы получить упражнения на неделю, используйте /advice"
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}
