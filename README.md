# Lovifyy Bot 🤖

Telegram бот с интеграцией искусственного интеллекта на языке Go.

## Описание

Этот бот использует локальную модель Ollama (Qwen 3:8B) для ответов на сообщения пользователей в Telegram. Бот может:
- Отвечать на любые вопросы через локальный ИИ
- Обрабатывать команды
- Поддерживать контекст разговора
- Работать стабильно на вашем сервере без внешних API
- Защищаться от спама и ограничивать частоту запросов

## Возможности

- 🤖 **ИИ-ответы**: Использует локальную модель Ollama (Qwen 3:8B) для генерации ответов
- 💬 **Команды**: Поддержка базовых команд (`/start`, `/help`, `/clear`, `/stats`)
- 🔄 **Асинхронная обработка**: Каждое сообщение обрабатывается в отдельной горутине
- 🛡️ **Безопасность**: Токены хранятся в переменных окружения, защита от спама
- ⏰ **Rate Limiting**: Ограничение частоты запросов (1 сообщение в 3 секунды)
- 📝 **Логирование**: Подробное логирование всех операций
- 🚫 **Валидация**: Проверка размера сообщений и защита от спама

## Установка и настройка

### 1. Клонирование проекта
```bash
git clone <your-repo-url>
cd Lovifyy_bot
```

### 2. Установка зависимостей
```bash
go mod tidy
```

### 3. Настройка переменных окружения

Создайте файл `.env` на основе `.env.example`:
```bash
cp .env.example .env
```

Отредактируйте `.env` файл и добавьте ваши токены:

#### Получение Telegram Bot Token:
1. Найдите [@BotFather](https://t.me/BotFather) в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям для создания бота
4. Скопируйте полученный токен в `.env`

#### Примечание:
OpenAI API больше не используется. Бот работает полностью локально с Ollama.

### 4. Установка и запуск Ollama

Скачайте и установите Ollama:
- Перейдите на https://ollama.com/download/windows
- Скачайте и установите OllamaSetup.exe
- Запустите Ollama
- Загрузите модель: `ollama pull qwen3:8b`

### 5. Запуск бота

**Простой способ:**
```bash
# Запуск через скрипт (рекомендуется)
scripts\run.bat
```

**Ручной запуск:**
```bash
go run cmd/main.go
```

**Сборка исполняемого файла:**
```bash
go build -o lovifyy_bot.exe cmd/main.go
./lovifyy_bot.exe
```

## Использование

### Команды бота:
- `/start` - Начать работу с ботом
- `/help` - Показать справку
- `/clear` - Очистить контекст разговора
- `/stats` - Показать статистику использования

### Обычные сообщения:
Просто напишите боту любое сообщение, и он ответит используя локальный ИИ!

### Ограничения:
- Максимум 4000 символов в сообщении
- 1 сообщение в 3 секунды (защита от спама)
- История ограничена 100 последними сообщениями

## Структура проекта

```
Lovifyy_bot/
├── cmd/
│   └── main.go              # Точка входа в приложение
├── internal/
│   ├── bot/
│   │   └── bot.go           # Основная логика Telegram бота
│   ├── ai/
│   │   └── ollama.go        # Клиент для работы с Ollama
│   └── history/
│       └── manager.go       # Управление историей сообщений
├── tests/
│   └── ollama_test.go       # Тесты для ИИ
├── configs/
│   └── .env.example         # Пример конфигурации
├── scripts/
│   ├── run.bat              # Скрипт запуска бота
│   └── test.bat             # Скрипт тестирования
├── chat_history/            # Папка с историей пользователей (создается автоматически)
├── go.mod                   # Зависимости Go
├── .env                     # Конфигурация (не в Git)
├── .gitignore              # Игнорируемые файлы
└── README.md               # Документация
```

## Развертывание на сервере

### Systemd сервис (Linux)

Создайте файл `/etc/systemd/system/lovifyy-bot.service`:

```ini
[Unit]
Description=Lovifyy Telegram Bot
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/Lovifyy_bot
ExecStart=/path/to/Lovifyy_bot/lovifyy_bot
Restart=always
RestartSec=10
Environment=PATH=/usr/bin:/usr/local/bin

[Install]
WantedBy=multi-user.target
```

Запустите сервис:
```bash
sudo systemctl daemon-reload
sudo systemctl enable lovifyy-bot
sudo systemctl start lovifyy-bot
```

### Docker (опционально)

Создайте `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o lovifyy_bot

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/lovifyy_bot .
CMD ["./lovifyy_bot"]
```

## Настройка и кастомизация

### Изменение модели ИИ
В файле `ai/ollama.go` измените модель при создании клиента:
```go
aiClient := ai.NewOllamaClient("llama2") // Вместо qwen3:8b
```

### Добавление новых команд
Добавьте новые case в функцию `handleCommand` в файле `bot.go`.

### Настройка системного промпта
Измените системное сообщение в функции `handleAIMessage` в файле `bot/bot.go` для изменения поведения ИИ.

## Мониторинг и логи

Бот логирует все важные события. Для просмотра логов в systemd:
```bash
sudo journalctl -u lovifyy-bot -f
```

## Безопасность

- ✅ Токены хранятся в переменных окружения
- ✅ `.env` файл добавлен в `.gitignore`
- ✅ Обработка ошибок для всех API вызовов
- ✅ Валидация входных данных
- ✅ Rate limiting (защита от спама)
- ✅ Ограничение размера сообщений
- ✅ HTTP таймауты для предотвращения зависания

## Требования

- Go 1.21+
- Telegram Bot Token
- Ollama с моделью Qwen 3:8B
- Локальное подключение к Ollama (порт 11434)

## Лицензия

MIT License

## Поддержка

Если у вас возникли вопросы или проблемы, создайте issue в репозитории.

---

Сделано с ❤️ для стабильной работы ИИ-бота в Telegram!
