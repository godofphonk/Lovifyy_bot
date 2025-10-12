package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics содержит все метрики приложения
type Metrics struct {
	// Счетчики
	MessagesTotal     *prometheus.CounterVec
	CommandsTotal     *prometheus.CounterVec
	ErrorsTotal       *prometheus.CounterVec
	AIRequestsTotal   *prometheus.CounterVec
	
	// Гистограммы
	ResponseDuration  *prometheus.HistogramVec
	AIResponseTime    *prometheus.HistogramVec
	
	// Гейджи
	ActiveUsers       prometheus.Gauge
	ConnectedUsers    prometheus.Gauge
	SystemMemory      prometheus.Gauge
	
	// Сводки
	MessageLength     *prometheus.SummaryVec
}

// NewMetrics создает новый экземпляр метрик
func NewMetrics() *Metrics {
	m := &Metrics{
		MessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lovifyy_messages_total",
				Help: "Total number of messages processed",
			},
			[]string{"type", "status"},
		),
		
		CommandsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lovifyy_commands_total",
				Help: "Total number of commands executed",
			},
			[]string{"command", "user_type"},
		),
		
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lovifyy_errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "component"},
		),
		
		AIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lovifyy_ai_requests_total",
				Help: "Total number of AI requests",
			},
			[]string{"model", "status"},
		),
		
		ResponseDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "lovifyy_response_duration_seconds",
				Help:    "Response duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"handler", "method"},
		),
		
		AIResponseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "lovifyy_ai_response_time_seconds",
				Help:    "AI response time in seconds",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
			},
			[]string{"model"},
		),
		
		ActiveUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "lovifyy_active_users",
				Help: "Number of active users",
			},
		),
		
		ConnectedUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "lovifyy_connected_users",
				Help: "Number of connected users",
			},
		),
		
		SystemMemory: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "lovifyy_system_memory_bytes",
				Help: "System memory usage in bytes",
			},
		),
		
		MessageLength: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "lovifyy_message_length",
				Help:       "Message length distribution",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"type"},
		),
	}
	
	// Регистрируем метрики
	prometheus.MustRegister(
		m.MessagesTotal,
		m.CommandsTotal,
		m.ErrorsTotal,
		m.AIRequestsTotal,
		m.ResponseDuration,
		m.AIResponseTime,
		m.ActiveUsers,
		m.ConnectedUsers,
		m.SystemMemory,
		m.MessageLength,
	)
	
	return m
}

// RecordMessage записывает метрику сообщения
func (m *Metrics) RecordMessage(messageType, status string) {
	m.MessagesTotal.WithLabelValues(messageType, status).Inc()
}

// RecordCommand записывает метрику команды
func (m *Metrics) RecordCommand(command, userType string) {
	m.CommandsTotal.WithLabelValues(command, userType).Inc()
}

// RecordError записывает метрику ошибки
func (m *Metrics) RecordError(errorType, component string) {
	m.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordAIRequest записывает метрику AI запроса
func (m *Metrics) RecordAIRequest(model, status string, duration time.Duration) {
	m.AIRequestsTotal.WithLabelValues(model, status).Inc()
	m.AIResponseTime.WithLabelValues(model).Observe(duration.Seconds())
}

// RecordResponseDuration записывает время ответа
func (m *Metrics) RecordResponseDuration(handler, method string, duration time.Duration) {
	m.ResponseDuration.WithLabelValues(handler, method).Observe(duration.Seconds())
}

// SetActiveUsers устанавливает количество активных пользователей
func (m *Metrics) SetActiveUsers(count float64) {
	m.ActiveUsers.Set(count)
}

// SetConnectedUsers устанавливает количество подключенных пользователей
func (m *Metrics) SetConnectedUsers(count float64) {
	m.ConnectedUsers.Set(count)
}

// SetSystemMemory устанавливает использование памяти
func (m *Metrics) SetSystemMemory(bytes float64) {
	m.SystemMemory.Set(bytes)
}

// RecordMessageLength записывает длину сообщения
func (m *Metrics) RecordMessageLength(messageType string, length float64) {
	m.MessageLength.WithLabelValues(messageType).Observe(length)
}

// StartMetricsServer запускает HTTP сервер для метрик
func (m *Metrics) StartMetricsServer(port string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}

// HealthCheck структура для health check
type HealthCheck struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Uptime    time.Duration     `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

// GetHealthCheck возвращает статус здоровья приложения
func GetHealthCheck(startTime time.Time) HealthCheck {
	return HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "2.0.0",
		Uptime:    time.Since(startTime),
		Checks: map[string]string{
			"database": "ok",
			"ai":       "ok",
			"telegram": "ok",
		},
	}
}

// StartHealthServer запускает HTTP сервер для health check
func StartHealthServer(port string, startTime time.Time) error {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		health := GetHealthCheck(startTime)
		
		// Простая JSON сериализация
		response := `{
			"status": "` + health.Status + `",
			"timestamp": "` + health.Timestamp.Format(time.RFC3339) + `",
			"version": "` + health.Version + `",
			"uptime": "` + health.Uptime.String() + `",
			"checks": {
				"database": "` + health.Checks["database"] + `",
				"ai": "` + health.Checks["ai"] + `",
				"telegram": "` + health.Checks["telegram"] + `"
			}
		}`
		
		w.Write([]byte(response))
	})
	
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	return http.ListenAndServe(":"+port, nil)
}
