#!/bin/bash

# Быстрое обновление бота на сервере
# Использование: ./update.sh

SERVER_HOST="192.168.0.102"
SERVER_USER="server"
PROJECT_NAME="lovifyy_bot"

echo "🔄 Обновляем Lovifyy Bot на сервере..."

# 1. Создаем архив только с измененными файлами
echo "📦 Создаем архив обновления..."
tar --exclude='.git' \
    --exclude='*.log' \
    --exclude='chat_history' \
    --exclude='diary_entries' \
    --exclude='scheduled_notifications.json' \
    --exclude='.env.server' \
    --exclude='.env.local' \
    -czf ${PROJECT_NAME}_update.tar.gz \
    internal/ cmd/ exercises/ *.go go.mod go.sum Dockerfile docker-compose.yml .env

# 2. Копируем и применяем обновление
echo "📤 Применяем обновление на сервере..."
scp ${PROJECT_NAME}_update.tar.gz ${SERVER_USER}@${SERVER_HOST}:~/

ssh ${SERVER_USER}@${SERVER_HOST} << 'EOF'
    cd /home/server/lovifyy_bot
    
    # Останавливаем бота
    echo "⏹️ Останавливаем бота..."
    docker-compose down
    
    # Применяем обновление
    echo "🔄 Применяем обновление..."
    tar -xzf ~/lovifyy_bot_update.tar.gz
    rm ~/lovifyy_bot_update.tar.gz
    
    # Пересобираем и запускаем
    echo "🔨 Пересобираем образ..."
    docker-compose build --no-cache lovifyy_bot
    
    echo "🚀 Запускаем бота..."
    docker-compose up -d
    
    echo "✅ Обновление завершено!"
    
    # Показываем статус
    sleep 3
    docker-compose ps
EOF

# 3. Очищаем локальный архив
rm ${PROJECT_NAME}_update.tar.gz

echo "✅ Обновление применено!"
echo "📊 Проверьте логи: ssh ${SERVER_USER}@${SERVER_HOST} 'cd /home/server/lovifyy_bot && docker-compose logs -f lovifyy_bot'"
