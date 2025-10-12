package admin

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleWelcome обрабатывает нажатие кнопки "Посмотреть приветствие"
func (h *Handler) HandleWelcome(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "👋 Текущее приветственное сообщение:\n\n" +
		"Привет, дорогие! 👋💖 Я так рад видеть вас здесь и вместе отправиться в это маленькое путешествие по вашим отношениям! 🫂\n\n" +
		"💡 Для изменения используйте:\n/setwelcome <новое приветствие>"
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}

// HandleSetWelcomeMenu обрабатывает нажатие кнопки "Изменить приветствие"
func (h *Handler) HandleSetWelcomeMenu(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "📝 Изменение приветственного сообщения\n\n" +
		"Отправьте команду в формате:\n" +
		"`/setwelcome <новое приветствие>`\n\n" +
		"💡 Приветствие отображается при команде /start"
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}
