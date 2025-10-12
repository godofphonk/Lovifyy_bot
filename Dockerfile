# Используем официальный образ Go
FROM golang:1.23-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Копируем .env файл если он существует
COPY .env* ./

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

# Создаем пользователя для безопасности
RUN adduser -D -s /bin/sh appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем бинарный файл из builder
COPY --from=builder /app/main .

# Копируем папку data с существующими файлами
COPY --from=builder /app/data ./data

# Создаем недостающие директории для данных
RUN mkdir -p data/chats data/diaries data/logs data/notifications data/backups && chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт (если понадобится в будущем)
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]
