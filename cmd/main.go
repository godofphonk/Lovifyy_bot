package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"Lovifyy_bot/internal/bot"
	"Lovifyy_bot/internal/config"
	"Lovifyy_bot/internal/logger"
	"Lovifyy_bot/internal/metrics"
	"Lovifyy_bot/internal/shutdown"
)

var (
	version   = "2.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	startTime := time.Now()

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Инициализируем логгер
	log := logger.NewLogger(cfg.Logger)
	log.WithFields(map[string]interface{}{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
	}).Info("Starting Lovifyy Bot")

	// Инициализируем метрики
	var metricsInstance *metrics.Metrics
	if cfg.Monitoring.Enabled {
		metricsInstance = metrics.NewMetrics()
		log.Info("Metrics system initialized")

		// Запускаем сервер метрик (используем порт health check + 1)
		go func() {
			port := fmt.Sprintf("%d", cfg.Monitoring.HealthCheckPort+1)
			log.WithField("port", port).Info("Starting metrics server")
			if err := metricsInstance.StartMetricsServer(port); err != nil {
				log.WithError(err).Error("Failed to start metrics server")
			}
		}()

		// Запускаем health check сервер
		go func() {
			port := fmt.Sprintf("%d", cfg.Monitoring.HealthCheckPort)
			log.WithField("port", port).Info("Starting health check server")
			if err := metrics.StartHealthServer(port, startTime); err != nil {
				log.WithError(err).Error("Failed to start health check server")
			}
		}()
	}

	// Инициализируем graceful shutdown
	shutdownManager := shutdown.NewPriorityManager(log, 30*time.Second)

	// Создаем контекст с отменой
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем бота
	telegramBot, err := bot.NewEnterpriseBot(cfg, log)
	if err != nil {
		log.WithError(err).Error("Failed to initialize enterprise bot")
		os.Exit(1)
	}

	// Регистрируем shutdown hooks
	shutdownManager.AddHook("telegram_bot", 100, func() error {
		log.Info("Shutting down Telegram bot")
		return telegramBot.Stop()
	})

	if metricsInstance != nil {
		shutdownManager.AddHook("metrics", 50, func() error {
			log.Info("Shutting down metrics system")
			// Здесь можно добавить логику остановки метрик
			return nil
		})
	}

	shutdownManager.AddHook("logger", 10, func() error {
		log.Info("Shutting down logger")
		return nil
	})

	// Запускаем бота в горутине
	go func() {
		log.Info("Starting enterprise Telegram bot")
		if err := telegramBot.Start(); err != nil {
			log.WithError(err).Error("Bot stopped with error")
		}
	}()

	// Логируем успешный запуск
	log.WithFields(map[string]interface{}{
		"startup_time": time.Since(startTime),
		"version":      version,
		"environment":  getEnvironment(),
	}).Info("Lovifyy Bot started successfully")

	// Обновляем метрики
	if metricsInstance != nil {
		metricsInstance.SetConnectedUsers(1)
		metricsInstance.RecordMessage("startup", "success")
	}

	// Ожидаем сигнал завершения
	shutdownManager.Wait()

	log.WithField("total_uptime", time.Since(startTime)).Info("Lovifyy Bot shutdown completed")
}

// getEnvironment определяет окружение
func getEnvironment() string {
	env := os.Getenv("GO_ENV")
	if env == "" {
		return "development"
	}
	return env
}
