#!/bin/bash

# Настройка прокси для доступа к OpenAI API
echo "🌐 Настраиваем прокси для OpenAI API..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Создаем docker-compose.yml с прокси
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  # SOCKS5 прокси контейнер
  proxy:
    image: serjs/go-socks5-proxy
    container_name: socks5_proxy
    ports:
      - "1080:1080"
    environment:
      - PROXY_USER=user
      - PROXY_PASSWORD=password
    restart: unless-stopped

  # Telegram бот с прокси
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    depends_on:
      - proxy
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=${SYSTEM_PROMPT}
      - ADMIN_IDS=${ADMIN_IDS}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - HTTP_PROXY=socks5://user:password@proxy:1080
      - HTTPS_PROXY=socks5://user:password@proxy:1080
      - ALL_PROXY=socks5://user:password@proxy:1080
    volumes:
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация создана"

# Запускаем контейнеры
echo "🚀 Запускаем контейнеры..."
docker-compose up -d

# Ждем запуска
sleep 10

# Проверяем статус
echo "📊 Статус контейнеров:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=10 lovifyy_bot

echo "✅ Прокси настроен! Проверьте логи бота."
