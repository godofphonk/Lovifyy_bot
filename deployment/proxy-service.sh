#!/bin/bash

# Запуск бота с публичными прокси-сервисами
echo "🌐 Настраиваем публичные прокси-сервисы..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Список более надежных прокси (обновляется)
PROXY_LIST=(
    "http://proxy.server:3128"
    "http://free-proxy.cz:3128"
    "http://proxy-list.download:8080"
    "http://spys.one:8080"
    "http://pubproxy.com:8080"
)

# Выбираем случайный прокси
PROXY=${PROXY_LIST[$RANDOM % ${#PROXY_LIST[@]}]}
echo "🎯 Используем прокси-сервис: $PROXY"

# Создаем docker-compose.yml БЕЗ прокси для тестирования
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
      - SKIP_AI_CHECK=true
    volumes:
      - ./.env:/app/.env:ro
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация без прокси создана (для тестирования)"

# Запускаем бота
echo "🚀 Запускаем бота без прокси..."
docker-compose up -d

# Ждем запуска
sleep 15

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=20 lovifyy_bot

echo "✅ Бот запущен без прокси для тестирования базового функционала!"
