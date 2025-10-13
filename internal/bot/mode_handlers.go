package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleChatMode обрабатывает команду /chat - активирует режим вопросов о отношениях
func (b *EnterpriseBot) handleChatMode(userID int64) error {
    // Создаем фейковый update для использования существующего обработчика
    update := tgbotapi.Update{
        CallbackQuery: &tgbotapi.CallbackQuery{
            Data: "chat",
            From: &tgbotapi.User{ID: userID},
            Message: &tgbotapi.Message{
                Chat: &tgbotapi.Chat{ID: userID},
            },
        },
    }
    
    // Используем существующий роутинг через HandleCallback
    return b.commandHandler.HandleCallback(update)
}

// handleDiaryMode обрабатывает команду /diary - показывает мини-дневник
func (b *EnterpriseBot) handleDiaryMode(userID int64) error {
    // Создаем фейковый update для использования существующего обработчика
    update := tgbotapi.Update{
        CallbackQuery: &tgbotapi.CallbackQuery{
            Data: "diary",
            From: &tgbotapi.User{ID: userID},
            Message: &tgbotapi.Message{
                Chat: &tgbotapi.Chat{ID: userID},
            },
        },
    }
    
    // Используем существующий роутинг через HandleCallback
    return b.commandHandler.HandleCallback(update)
}

// handleExercises обрабатывает команду /advice - показывает упражнения недели
func (b *EnterpriseBot) handleExercises(userID int64) error {
    // Создаем фейковый update для использования существующего обработчика
    update := tgbotapi.Update{
        CallbackQuery: &tgbotapi.CallbackQuery{
            Data: "advice",
            From: &tgbotapi.User{ID: userID},
            Message: &tgbotapi.Message{
                Chat: &tgbotapi.Chat{ID: userID},
            },
        },
    }
    
    // Используем существующий роутинг через HandleCallback
    return b.commandHandler.HandleCallback(update)
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
