#!/bin/bash

# Автоматическое исправление бота на сервере
echo "🔧 Исправляем конфигурацию бота на сервере..."

# Переходим в правильную папку
cd /home/server/lovifyy_bot

# Останавливаем все контейнеры
echo "⏹️ Останавливаем контейнеры..."
docker-compose down 2>/dev/null || true

# Создаем правильный docker-compose.yml
echo "📝 Создаем новый docker-compose.yml..."
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot_warp
    network_mode: "container:warp"
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

# Проверяем .env файл
if [ ! -f .env ]; then
    echo "❌ .env файл не найден!"
    echo "Скопируйте .env файл командой:"
    echo "scp .env server@192.168.0.102:/home/server/lovifyy_bot/"
    exit 1
fi

# Запускаем бота
echo "🚀 Запускаем бота..."
docker-compose up -d

# Ждем немного
sleep 5

# Показываем статус
echo "📊 Статус контейнеров:"
docker ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=20 lovifyy_bot_warp

echo "✅ Готово! Бот должен работать через WARP VPN"
