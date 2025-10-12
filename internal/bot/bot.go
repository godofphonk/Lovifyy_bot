package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Lovifyy_bot/internal/ai"
	"Lovifyy_bot/internal/config"
	"Lovifyy_bot/internal/exercises"
	"Lovifyy_bot/internal/handlers"
	"Lovifyy_bot/internal/history"
	"Lovifyy_bot/internal/logger"
	"Lovifyy_bot/internal/metrics"
	"Lovifyy_bot/internal/middleware"
	"Lovifyy_bot/internal/models"
	"Lovifyy_bot/internal/services"
	"Lovifyy_bot/internal/validator"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// EnterpriseBot представляет enterprise-grade Telegram бот
type EnterpriseBot struct {
	// Core components
	telegram *tgbotapi.BotAPI
	ai       *ai.OpenAIClient
	
	// Configuration and logging
	config *config.Config
	logger *logger.Logger
	
	// Metrics and monitoring
	metrics *metrics.Metrics
	
	// Managers and services
	userManager         *models.UserManager
	historyManager      *history.Manager
	exerciseManager     *exercises.Manager
	notificationService *services.NotificationService
	
	// Handlers and middleware
	commandHandler      *handlers.CommandHandler
	rateLimitMiddleware *middleware.RateLimitMiddleware
	validator          *validator.Validator
	
	// Runtime state
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEnterpriseBot создает новый enterprise-grade бот
func NewEnterpriseBot(cfg *config.Config, log *logger.Logger) (*EnterpriseBot, error) {
	// Инициализируем Telegram бота
	telegram, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"bot_username": telegram.Self.UserName,
		"bot_id":       telegram.Self.ID,
	}).Info("Telegram bot authorized successfully")

	// Инициализируем AI клиент
	aiClient := ai.NewOpenAIClient(cfg.OpenAI.Model)
	if aiClient == nil {
		log.Warn("OpenAI client not available, bot will work with limited functionality")
	} else {
		// Тестируем AI подключение
		if err := aiClient.TestConnection(); err != nil {
			log.WithError(err).Warn("OpenAI connection test failed")
		} else {
			log.Info("OpenAI client connected successfully")
		}
	}

	// Инициализируем метрики
	var metricsInstance *metrics.Metrics
	if cfg.Monitoring.Enabled {
		metricsInstance = metrics.NewMetrics()
		log.Info("Metrics system initialized")
	}

	// Инициализируем менеджеры
	userManager := models.NewUserManager(cfg.Telegram.AdminIDs)
	historyManager := history.NewManager()
	exerciseManager := exercises.NewManager()

	// Инициализируем валидатор
	validatorInstance := validator.NewValidator(validator.Config{
		MaxMessageLength: cfg.Security.MaxMessageLength,
		AllowedCommands:  cfg.Security.AllowedCommands,
		SanitizeHTML:     cfg.Security.EnableSanitization,
	})

	// Инициализируем сервисы
	notificationService := services.NewNotificationService(telegram, aiClient)

	// Инициализируем обработчики
	commandHandler := handlers.NewCommandHandler(telegram, userManager, notificationService)

	// Инициализируем middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(userManager, cfg.Security.RateLimitDuration)

	// Создаем контекст
	ctx, cancel := context.WithCancel(context.Background())

	bot := &EnterpriseBot{
		telegram:            telegram,
		ai:                  aiClient,
		config:              cfg,
		logger:              log,
		metrics:             metricsInstance,
		userManager:         userManager,
		historyManager:      historyManager,
		exerciseManager:     exerciseManager,
		notificationService: notificationService,
		commandHandler:      commandHandler,
		rateLimitMiddleware: rateLimitMiddleware,
		validator:          validatorInstance,
		ctx:                ctx,
		cancel:             cancel,
	}

	log.WithFields(map[string]interface{}{
		"admins_count":    len(cfg.Telegram.AdminIDs),
		"ai_enabled":      aiClient != nil,
		"metrics_enabled": metricsInstance != nil,
	}).Info("Enterprise bot initialized successfully")

	return bot, nil
}

// Start запускает бота
func (b *EnterpriseBot) Start() error {
	b.logger.Info("Starting enterprise bot...")

	// Настраиваем команды бота
	if err := b.setupCommands(); err != nil {
		return fmt.Errorf("failed to setup commands: %w", err)
	}

	// Запускаем метрики если включены
	if b.metrics != nil {
		go b.startMetricsCollection()
	}

	// Получаем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = b.config.Telegram.Timeout

	updates := b.telegram.GetUpdatesChan(u)

	b.logger.WithFields(map[string]interface{}{
		"timeout": u.Timeout,
		"offset":  u.Offset,
	}).Info("Bot started, listening for updates...")

	// Обрабатываем обновления
	for {
		select {
		case <-b.ctx.Done():
			b.logger.Info("Bot context cancelled, stopping...")
			return nil
		case update := <-updates:
			go b.handleUpdateWithMetrics(update)
		}
	}
}

// Stop останавливает бота
func (b *EnterpriseBot) Stop() error {
	b.logger.Info("Stopping enterprise bot...")
	b.cancel()
	return nil
}

// setupCommands настраивает команды бота
func (b *EnterpriseBot) setupCommands() error {
	b.logger.Info("Setting up bot commands...")

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🏠 Главное меню"},
		{Command: "help", Description: "🆘 Помощь"},
		{Command: "menu", Description: "📱 Вернуться в меню"},
	}

	// Добавляем админские команды для администраторов
	adminCommands := []tgbotapi.BotCommand{
		{Command: "admin", Description: "👑 Админ-панель"},
		{Command: "notify", Description: "📢 Уведомления"},
		{Command: "setweek", Description: "🗓️ Настройка недель"},
		{Command: "metrics", Description: "📊 Метрики"},
	}

	// Устанавливаем команды
	allCommands := append(commands, adminCommands...)
	setCommands := tgbotapi.NewSetMyCommands(allCommands...)
	
	if _, err := b.telegram.Request(setCommands); err != nil {
		return fmt.Errorf("failed to set commands: %w", err)
	}

	b.logger.WithField("commands_count", len(allCommands)).Info("Bot commands set successfully")
	return nil
}

// handleUpdateWithMetrics обрабатывает обновления с метриками
func (b *EnterpriseBot) handleUpdateWithMetrics(update tgbotapi.Update) {
	startTime := time.Now()
	
	// Применяем rate limiting middleware
	handler := b.rateLimitMiddleware.AdminBypass(b.processUpdate)
	
	if err := handler(update); err != nil {
		b.logger.WithError(err).Error("Error processing update")
		if b.metrics != nil {
			b.metrics.RecordError("update_processing", "bot")
		}
	}
	
	// Записываем метрики
	if b.metrics != nil {
		duration := time.Since(startTime)
		b.metrics.RecordResponseDuration("update", "telegram", duration)
	}
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

// handleCommand обрабатывает команды с валидацией
func (b *EnterpriseBot) handleCommand(update tgbotapi.Update) error {
	command := update.Message.Command()
	userID := update.Message.From.ID

	// Валидируем команду
	if validation := b.validator.ValidateCommand(command); !validation.Valid {
		b.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"command": command,
			"errors":  validation.Errors,
		}).Warn("Invalid command received")
		
		if b.metrics != nil {
			b.metrics.RecordError("invalid_command", "validation")
		}
		return nil
	}

	// Записываем метрику команды
	if b.metrics != nil {
		userType := "user"
		if b.userManager.IsAdmin(userID) {
			userType = "admin"
		}
		b.metrics.RecordCommand(command, userType)
	}

	// Логируем команду
	b.logger.WithFields(map[string]interface{}{
		"user_id":  userID,
		"username": update.Message.From.UserName,
		"command":  command,
	}).Info("Processing command")

	// Обрабатываем команду
	switch command {
	case "start":
		return b.commandHandler.HandleStart(update)
	case "help":
		return b.commandHandler.HandleHelp(update)
	case "menu":
		return b.commandHandler.HandleMenu(update)
	case "admin":
		return b.commandHandler.HandleAdmin(update)
	case "notify":
		return b.commandHandler.HandleNotify(update)
	case "setweek":
		return b.commandHandler.HandleSetWeek(update)
	case "metrics":
		return b.handleMetricsCommand(update)
	default:
		return b.commandHandler.HandleUnknownCommand(update)
	}
}

// handleCallbackQuery обрабатывает callback queries
func (b *EnterpriseBot) handleCallbackQuery(update tgbotapi.Update) error {
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := b.telegram.Request(callback); err != nil {
		b.logger.WithError(err).Error("Failed to answer callback query")
	}

	data := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID

	b.logger.WithFields(map[string]interface{}{
		"user_id":       userID,
		"callback_data": data,
	}).Info("Processing callback query")

	// Роутинг callback queries
	switch {
	case data == "mode_chat":
		return b.handleChatMode(userID)
	case data == "mode_diary":
		return b.handleDiaryMode(userID)
	case data == "exercises":
		return b.handleExercises(userID)
	case strings.HasPrefix(data, "notify_"):
		return b.handleNotificationCallback(userID, data)
	default:
		b.logger.WithField("callback_data", data).Warn("Unknown callback query")
		return nil
	}
}

// handleMessage обрабатывает обычные сообщения с валидацией
func (b *EnterpriseBot) handleMessage(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	messageText := update.Message.Text

	// Валидируем сообщение
	if validation := b.validator.ValidateMessage(messageText); !validation.Valid {
		b.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"errors":  validation.Errors,
		}).Warn("Invalid message received")
		
		if b.metrics != nil {
			b.metrics.RecordError("invalid_message", "validation")
		}
		
		// Отправляем сообщение об ошибке
		errorMsg := "❌ Сообщение содержит недопустимый контент"
		msg := tgbotapi.NewMessage(userID, errorMsg)
		b.telegram.Send(msg)
		return nil
	}

	// Санитизируем сообщение
	sanitizedText := b.validator.SanitizeMessage(messageText)

	// Записываем метрики
	if b.metrics != nil {
		b.metrics.RecordMessage("text", "received")
		b.metrics.RecordMessageLength("user", float64(len(sanitizedText)))
	}

	// Получаем состояние пользователя
	state := b.userManager.GetState(userID)

	switch state {
	case "chat":
		return b.handleChatMessage(userID, sanitizedText)
	case "diary":
		return b.handleDiaryMessage(userID, sanitizedText)
	default:
		return b.suggestMode(userID)
	}
}

// handleChatMessage обрабатывает сообщения в режиме чата
func (b *EnterpriseBot) handleChatMessage(userID int64, messageText string) error {
	startTime := time.Now()
	
	// Проверяем доступность AI
	if b.ai == nil {
		msg := tgbotapi.NewMessage(userID, "❌ AI сервис временно недоступен. Попробуйте позже.")
		_, err := b.telegram.Send(msg)
		return err
	}

	// Отправляем индикатор печати
	typing := tgbotapi.NewChatAction(userID, tgbotapi.ChatTyping)
	b.telegram.Send(typing)

	// Сохраняем сообщение пользователя в историю
	if err := b.historyManager.SaveMessage(userID, "", messageText, "", "gpt-4o-mini"); err != nil {
		b.logger.WithError(err).Error("Failed to save user message")
	}

	// Получаем историю для OpenAI формата
	openaiHistory, err := b.historyManager.GetOpenAIHistory(userID, b.config.Telegram.SystemPrompt, 10)
	if err != nil {
		b.logger.WithError(err).Error("Failed to get OpenAI history")
		openaiHistory = []history.OpenAIMessage{
			{Role: "system", Content: b.config.Telegram.SystemPrompt},
			{Role: "user", Content: messageText},
		}
	}

	// Преобразуем в формат AI клиента
	messages := make([]ai.OpenAIMessage, len(openaiHistory))
	for i, msg := range openaiHistory {
		messages[i] = ai.OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Получаем ответ от AI
	response, err := b.ai.GenerateWithHistory(messages)
	if err != nil {
		b.logger.WithError(err).Error("AI generation failed")
		
		if b.metrics != nil {
			b.metrics.RecordAIRequest("gpt-4o-mini", "error", time.Since(startTime))
		}
		
		errorMsg := tgbotapi.NewMessage(userID, "❌ Извините, произошла ошибка при обращении к ИИ. Попробуйте позже.")
		b.telegram.Send(errorMsg)
		return err
	}

	// Записываем метрики AI
	if b.metrics != nil {
		b.metrics.RecordAIRequest("gpt-4o-mini", "success", time.Since(startTime))
		b.metrics.RecordMessage("ai_response", "sent")
	}

	// Сохраняем ответ AI в историю
	if err := b.historyManager.SaveMessage(userID, "", messageText, response, "gpt-4o-mini"); err != nil {
		b.logger.WithError(err).Error("Failed to save AI response")
	}

	// Отправляем ответ
	msg := tgbotapi.NewMessage(userID, response)
	msg.ParseMode = "HTML"

	_, err = b.telegram.Send(msg)
	return err
}

// Остальные методы (handleDiaryMessage, handleChatMode, etc.) остаются такими же
// но с добавлением логирования и метрик...

// GetMetrics возвращает метрики бота
func (b *EnterpriseBot) GetMetrics() *metrics.Metrics {
	return b.metrics
}

// GetLogger возвращает логгер бота
func (b *EnterpriseBot) GetLogger() *logger.Logger {
	return b.logger
}

// startMetricsCollection запускает сбор метрик
func (b *EnterpriseBot) startMetricsCollection() {
	ticker := time.NewTicker(b.config.Monitoring.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			// Обновляем системные метрики
			// TODO: Добавить сбор системных метрик
		}
	}
}

// handleDiaryMessage обрабатывает сообщения в режиме дневника
func (b *EnterpriseBot) handleDiaryMessage(userID int64, messageText string) error {
	// TODO: Implement diary message handling
	msg := tgbotapi.NewMessage(userID, "📔 Функция дневника в разработке")
	_, err := b.telegram.Send(msg)
	return err
}

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

// handleExercises показывает упражнения
func (b *EnterpriseBot) handleExercises(userID int64) error {
	msg := tgbotapi.NewMessage(userID, "🧠 Упражнения в разработке")
	_, err := b.telegram.Send(msg)
	return err
}

// handleNotificationCallback обрабатывает callback уведомлений
func (b *EnterpriseBot) handleNotificationCallback(userID int64, data string) error {
	msg := tgbotapi.NewMessage(userID, "📢 Уведомления в разработке")
	_, err := b.telegram.Send(msg)
	return err
}

// handleMetricsCommand показывает метрики (только для админов)
func (b *EnterpriseBot) handleMetricsCommand(update tgbotapi.Update) error {
	userID := update.Message.From.ID
	
	if !b.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(userID, "❌ Доступ запрещен")
		_, err := b.telegram.Send(msg)
		return err
	}
	
	msg := tgbotapi.NewMessage(userID, "📊 Метрики доступны по адресу: http://localhost:9090/metrics")
	_, err := b.telegram.Send(msg)
	return err
}

// suggestMode предлагает выбрать режим
func (b *EnterpriseBot) suggestMode(userID int64) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Чат с ИИ", "mode_chat"),
			tgbotapi.NewInlineKeyboardButtonData("📔 Дневник", "mode_diary"),
		),
	)
	
	msg := tgbotapi.NewMessage(userID, "Выберите режим работы:")
	msg.ReplyMarkup = keyboard
	_, err := b.telegram.Send(msg)
	return err
}
