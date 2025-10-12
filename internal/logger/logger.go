package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger представляет структурированный логгер
type Logger struct {
	*logrus.Logger
}

// Config конфигурация логгера
type Config struct {
	Level      string `json:"level"`
	Format     string `json:"format"`     // json, text
	Output     string `json:"output"`     // stdout, file
	Filename   string `json:"filename"`
	MaxSize    int    `json:"max_size"`   // MB
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`    // days
}

// NewLogger создает новый структурированный логгер
func NewLogger(config Config) *Logger {
	log := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Устанавливаем формат
	switch strings.ToLower(config.Format) {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Устанавливаем вывод
	switch strings.ToLower(config.Output) {
	case "file":
		if config.Filename != "" {
			file, err := os.OpenFile(config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				log.SetOutput(file)
			}
		}
	default:
		log.SetOutput(os.Stdout)
	}

	return &Logger{Logger: log}
}

// GetDefaultConfig возвращает конфигурацию по умолчанию
func GetDefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "text",
		Output:     "stdout",
		Filename:   "data/logs/lovifyy_bot.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
	}
}

// WithFields добавляет поля к логу
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithUserID добавляет ID пользователя к логу
func (l *Logger) WithUserID(userID int64) *logrus.Entry {
	return l.Logger.WithField("user_id", userID)
}

// WithError добавляет ошибку к логу
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// LogUserAction логирует действие пользователя
func (l *Logger) LogUserAction(userID int64, action, details string) {
	l.WithFields(map[string]interface{}{
		"user_id": userID,
		"action":  action,
		"details": details,
	}).Info("User action")
}

// LogAPICall логирует вызов API
func (l *Logger) LogAPICall(service, method string, duration int64, success bool) {
	l.WithFields(map[string]interface{}{
		"service":  service,
		"method":   method,
		"duration": duration,
		"success":  success,
	}).Info("API call")
}

// LogError логирует ошибку с контекстом
func (l *Logger) LogError(err error, context string, fields map[string]interface{}) {
	entry := l.WithError(err).WithField("context", context)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("Error occurred")
}

// LogMetric логирует метрику
func (l *Logger) LogMetric(name string, value interface{}, tags map[string]string) {
	fields := map[string]interface{}{
		"metric": name,
		"value":  value,
	}
	for k, v := range tags {
		fields["tag_"+k] = v
	}
	l.WithFields(fields).Info("Metric")
}
