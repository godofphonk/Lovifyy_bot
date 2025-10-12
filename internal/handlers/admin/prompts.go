package admin

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandlePrompt обрабатывает нажатие кнопки "Посмотреть промпт"
func (h *Handler) HandlePrompt(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "🤖 Текущий системный промпт:\n\n" +
		"Ты опытный психолог и консультант по отношениям и личностному росту пар и людей.\n\n" +
		"💡 Для изменения используйте:\n/setprompt <новый промпт>"
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}

// HandleSetPromptMenu обрабатывает нажатие кнопки "Изменить промпт"
func (h *Handler) HandleSetPromptMenu(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта команда доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	response := "✏️ Изменение системного промпта\n\n" +
		"Отправьте команду в формате:\n" +
		"`/setprompt <новый промпт>`\n\n" +
		"💡 Готовые варианты:\n\n" +
		"Психолог:\n" +
		"`/setprompt Ты опытный психолог, который помогает людям с их личными проблемами. Будь сочувствующим и давай полезные советы.`\n\n" +
		"Дружелюбный помощник:\n" +
		"`/setprompt Ты дружелюбный помощник, готовый ответить на любые вопросы. Будь позитивным и полезным.`"
	
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err := h.bot.Send(msg)
	return err
}
