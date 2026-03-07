package bot

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCommand обрабатывает команды с валидацией
func (b *EnterpriseBot) handleCommand(update tgbotapi.Update) error {
	command := update.Message.Command()
	userID := update.Message.From.ID

	b.logger.WithFields(map[string]interface{}{
		"user_id":  userID,
		"username": update.Message.From.UserName,
		"command":  command,
	}).Info("Processing command")

	// Записываем метрики
	if b.metrics != nil {
		b.metrics.RecordCommand(command, "telegram")
	}

	// Обрабатываем команды
	switch command {
	case "start":
		return b.commandHandler.HandleStart(update)
	case "help":
		return b.commandHandler.HandleHelp(update)
	case "chat":
		return b.handleChatMode(userID)
	case "diary":
		return b.handleDiaryMode(userID)
	case "advice":
		return b.handleExercises(userID)
	case "setweek":
		return b.commandHandler.HandleSetWeek(update)
	case "adminhelp":
		return b.commandHandler.HandleAdmin(update)
	case "metrics":
		return b.handleMetricsCommand(update)
	default:
		msg := tgbotapi.NewMessage(userID, "❓ Неизвестная команда. Используйте /help для справки.")
		_, err := b.telegram.Send(msg)
		return err
	}
}

// handleMetricsCommand показывает метрики (только для админов)
func (b *EnterpriseBot) handleMetricsCommand(update tgbotapi.Update) error {
	userID := update.Message.From.ID

	if !b.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ У вас нет прав для просмотра метрик")
		_, err := b.telegram.Send(msg)
		return err
	}

	// TODO: Implement metrics display
	msg := tgbotapi.NewMessage(userID, "📊 Метрики бота в разработке")
	_, err := b.telegram.Send(msg)
	return err
}

// setupCommands настраивает команды бота
func (b *EnterpriseBot) setupCommands() error {
	b.logger.Info("Setting up bot commands...")

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🚀 Начать работу с ботом"},
		{Command: "advice", Description: "💑 Упражнение недели"},
		{Command: "diary", Description: "📝 Мини-дневник"},
		{Command: "chat", Description: "💒 Задать вопрос о отношениях"},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := b.telegram.Request(config)
	if err != nil {
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	b.logger.WithField("commands_count", len(commands)).Info("Bot commands set successfully")
	return nil
}

// processUpdate основная логика обработки обновлений
func (b *EnterpriseBot) processUpdate(update tgbotapi.Update) error {
	// Обрабатываем команды
	if update.Message != nil && update.Message.IsCommand() {
		return b.handleCommand(update)
	}

	// Обрабатываем callback queries
	if update.CallbackQuery != nil {
		return b.handleCallbackQuery(update)
	}

	// Обрабатываем обычные сообщения
	if update.Message != nil {
		return b.handleMessage(update)
	}

	return nil
}

// handleUpdateWithMetrics обрабатывает обновления с метриками
func (b *EnterpriseBot) handleUpdateWithMetrics(update tgbotapi.Update) {
	startTime := time.Now()

	// Обрабатываем обновление
	if err := b.processUpdate(update); err != nil {
		b.logger.WithError(err).Error("Failed to process update")

		if b.metrics != nil {
			b.metrics.RecordError("update_processing", "general")
		}
	}

	// Записываем метрики времени обработки
	if b.metrics != nil {
		duration := time.Since(startTime)
		b.metrics.RecordResponseDuration("update_processing", "telegram", duration)
	}
}
