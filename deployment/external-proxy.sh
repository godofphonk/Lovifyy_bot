#!/bin/bash

# Настройка с внешним прокси для OpenAI API
echo "🌐 Настраиваем внешний прокси для OpenAI API..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Список бесплатных SOCKS5 прокси (обновляется)
PROXY_LIST=(
    "socks5://198.8.94.174:4145"
    "socks5://72.210.221.223:4145"
    "socks5://184.178.172.25:15291"
    "socks5://199.58.185.9:4145"
    "socks5://72.195.34.59:4145"
)

# Выбираем случайный прокси
PROXY=${PROXY_LIST[$RANDOM % ${#PROXY_LIST[@]}]}
echo "🎯 Используем прокси: $PROXY"

# Создаем docker-compose.yml с внешним прокси
cat > docker-compose.yml << EOF
version: '3.8'

services:
  # Telegram бот с внешним прокси
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    environment:
      - TELEGRAM_BOT_TOKEN=\${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=\${SYSTEM_PROMPT}
      - ADMIN_IDS=\${ADMIN_IDS}
      - OPENAI_API_KEY=\${OPENAI_API_KEY}
      - HTTP_PROXY=$PROXY
      - HTTPS_PROXY=$PROXY
      - ALL_PROXY=$PROXY
    volumes:
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
sleep 10

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=15 lovifyy_bot

echo "✅ Внешний прокси настроен! Если не работает, запустите скрипт еще раз для смены прокси."
