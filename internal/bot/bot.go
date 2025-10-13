package bot

import (
    "context"
    "fmt"
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

	telegram.Debug = false // Устанавливаем debug режим
	log.WithField("bot_username", telegram.Self.UserName).Info("Telegram bot authorized successfully")

	// Инициализируем AI клиент
	aiClient := ai.NewOpenAIClient("gpt-4o-mini")

	// Инициализируем менеджеры
	userManager := models.NewUserManager([]int64{1805441944, 1243795198}) // Список админов
	historyManager := history.NewManager()
	exerciseManager := exercises.NewManager()
	
	// Инициализируем сервисы
	notificationService := services.NewNotificationService(telegram, aiClient)
	
	// Инициализируем middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(userManager, time.Minute)
	validator := validator.NewValidator(validator.Config{
		MaxMessageLength: 4000, // Увеличиваем лимит сообщений
		SanitizeHTML:     true,
	})
	
	// Инициализируем метрики
	var metricsInstance *metrics.Metrics
	metricsInstance = metrics.NewMetrics()

	// Создаем контекст
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем бота
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
		rateLimitMiddleware: rateLimitMiddleware,
		validator:          validator,
		ctx:                ctx,
		cancel:             cancel,
	}

	// Инициализируем обработчик команд
	bot.commandHandler = handlers.NewCommandHandler(
		telegram, userManager, exerciseManager, notificationService, historyManager, aiClient,
	)

	return bot, nil
}

// Start запускает бота
func (b *EnterpriseBot) Start() error {
	b.logger.Info("Starting enterprise bot...")

	// Настраиваем команды бота
	if err := b.setupCommands(); err != nil {
		return fmt.Errorf("failed to setup commands: %w", err)
	}

	// Запускаем сбор метрик
	if b.metrics != nil {
		go b.startMetricsCollection()
	}

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.telegram.GetUpdatesChan(u)

	b.logger.WithFields(map[string]interface{}{
		"offset":  u.Offset,
		"timeout": u.Timeout,
	}).Info("Bot started, listening for updates...")

	// Обрабатываем обновления
	for {
		select {
		case update := <-updates:
			go b.handleUpdateWithMetrics(update)
		case <-b.ctx.Done():
			return nil
		}
	}
}

// Stop останавливает бота
func (b *EnterpriseBot) Stop() error {
	b.logger.Info("Stopping enterprise bot...")
	b.cancel()
	return nil
}
