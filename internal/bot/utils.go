package bot

import (
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/metrics"
	"github.com/godofphonk/lovifyy-bot/internal/logger"
)

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
		case <-ticker.C:
			// Собираем метрики
			// TODO: Implement metrics collection
		case <-b.ctx.Done():
			return
		}
	}
}
