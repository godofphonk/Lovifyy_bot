#!/bin/bash

# Простое решение - тестируем бота через публичные прокси
echo "🌐 Тестируем бота с публичными прокси..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Список рабочих прокси (обновляется)
PROXY_LIST=(
    "http://proxy.server:3128"
    "http://47.74.152.29:8888"
    "http://103.149.162.194:80"
    "http://185.162.251.76:80"
    ""  # Без прокси для сравнения
)

# Выбираем случайный прокси
PROXY=${PROXY_LIST[$RANDOM % ${#PROXY_LIST[@]}]}

if [ -z "$PROXY" ]; then
    echo "🎯 Тестируем БЕЗ прокси"
    PROXY_ENV=""
else
    echo "🎯 Используем прокси: $PROXY"
    PROXY_ENV="
      - HTTP_PROXY=$PROXY
      - HTTPS_PROXY=$PROXY
      - ALL_PROXY=$PROXY"
fi

# Создаем простую конфигурацию
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
      - OPENAI_API_KEY=\${OPENAI_API_KEY}$PROXY_ENV
    volumes:
      - ./.env:/app/.env:ro
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация создана"

# Запускаем бота
echo "🚀 Запускаем бота..."
docker-compose up -d

# Ждем запуска
sleep 15

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота (последние 20 строк):"
docker-compose logs --tail=20 lovifyy_bot

echo ""
echo "🔍 Проверяем работу бота..."
if docker-compose ps | grep -q "Up"; then
    echo "✅ Контейнер запущен!"
    
    # Проверяем логи на ошибки
    if docker logs lovifyy_bot 2>&1 | grep -q "Авторизован как"; then
        echo "✅ Telegram подключение работает!"
    fi
    
    if docker logs lovifyy_bot 2>&1 | grep -q "AI недоступен"; then
        echo "⚠️  AI недоступен (ожидаемо без VPN)"
    fi
    
    if docker logs lovifyy_bot 2>&1 | grep -q "403"; then
        echo "❌ OpenAI API заблокирован"
    fi
else
    echo "❌ Контейнер не запустился"
fi

echo ""
echo "🎯 Результат теста с прокси: $PROXY"
