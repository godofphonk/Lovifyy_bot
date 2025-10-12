#!/bin/bash

# Запуск бота без прокси (для тестирования базового функционала)
echo "🚀 Запускаем бота без прокси..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Создаем простой docker-compose.yml без прокси
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  # Telegram бот без прокси
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=${SYSTEM_PROMPT}
      - ADMIN_IDS=${ADMIN_IDS}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация без прокси создана"

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

echo "✅ Бот запущен без прокси! Telegram функции должны работать, AI будет недоступен."
