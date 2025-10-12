package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleChatMode переключает в режим чата
func (b *EnterpriseBot) handleChatMode(userID int64) error {
	b.userManager.SetState(userID, "chat")
	msg := tgbotapi.NewMessage(userID, "💬 Режим чата активирован! Можете задавать вопросы.")
	_, err := b.telegram.Send(msg)
	return err
}

// handleDiaryMode переключает в режим дневника
func (b *EnterpriseBot) handleDiaryMode(userID int64) error {
    b.userManager.SetState(userID, "diary")
    msg := tgbotapi.NewMessage(userID, "📔 Режим дневника активирован! Пишите свои мысли.")
    _, err := b.telegram.Send(msg)
    return err
}

// handleExercises больше не используется: показ «Упражнений» делегирован в CommandHandler
func (b *EnterpriseBot) handleExercises(userID int64) error {
    // На всякий случай, предложим выбрать режим
    return b.suggestMode(userID)
}

// suggestMode предлагает выбрать режим
func (b *EnterpriseBot) suggestMode(userID int64) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💒 Задать вопрос о отношениях", "mode_chat"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👩🏼‍❤️‍👨🏻 Упражнение недели", "exercises"),
			tgbotapi.NewInlineKeyboardButtonData("💌 Мини-дневник", "mode_diary"),
		),
	)
	
	msg := tgbotapi.NewMessage(userID, "Выберите режим работы:")
	msg.ReplyMarkup = keyboard
	_, err := b.telegram.Send(msg)
	return err
}
