#!/bin/bash

# Запуск бота с бесплатными прокси (ротация)
echo "🌐 Настраиваем бесплатные прокси для OpenAI API..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Список бесплатных HTTP прокси (обновляется ежедневно)
PROXY_LIST=(
    "http://47.74.152.29:8888"
    "http://103.149.162.194:80"
    "http://185.162.251.76:80"
    "http://47.88.3.19:8080"
    "http://103.167.71.20:80"
    "http://47.74.152.29:8888"
    "http://198.49.68.80:80"
    "http://103.149.162.195:80"
)

# Выбираем случайный прокси
PROXY=${PROXY_LIST[$RANDOM % ${#PROXY_LIST[@]}]}
echo "🎯 Используем прокси: $PROXY"

# Создаем docker-compose.yml с прокси
cat > docker-compose.yml << EOF
version: '3.8'

services:
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    env_file:
      - .env
    environment:
      - TELEGRAM_BOT_TOKEN=\${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=\${SYSTEM_PROMPT}
      - ADMIN_IDS=\${ADMIN_IDS}
      - OPENAI_API_KEY=\${OPENAI_API_KEY}
      - HTTP_PROXY=$PROXY
      - HTTPS_PROXY=$PROXY
      - ALL_PROXY=$PROXY
    volumes:
      - ./.env:/app/.env:ro
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация с прокси $PROXY создана"

# Запускаем бота
echo "🚀 Запускаем бота..."
docker-compose up -d

# Ждем запуска
sleep 15

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=20 lovifyy_bot

echo "✅ Бесплатный прокси настроен! Если не работает, запустите скрипт еще раз для смены прокси."
