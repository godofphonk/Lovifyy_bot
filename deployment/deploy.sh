#!/bin/bash

# Деплой Lovifyy Bot на удаленный сервер
# Использование: ./deploy.sh

SERVER_HOST="192.168.0.102"
SERVER_USER="server"
PROJECT_NAME="lovifyy_bot"
REMOTE_PATH="/home/server/lovifyy_bot"

echo "🚀 Начинаем деплой Lovifyy Bot на сервер $SERVER_HOST..."

# 1. Создаем архив проекта (исключая ненужные файлы)
echo "📦 Создаем архив проекта..."
tar --exclude='.git' \
    --exclude='*.log' \
    --exclude='chat_history' \
    --exclude='diary_entries' \
    --exclude='scheduled_notifications.json' \
    --exclude='node_modules' \
    --exclude='.env.local' \
    --exclude='.env.server' \
    -czf ${PROJECT_NAME}.tar.gz .

# 2. Копируем архив на сервер
echo "📤 Копируем файлы на сервер..."
scp ${PROJECT_NAME}.tar.gz ${SERVER_USER}@${SERVER_HOST}:~/

# 3. Подключаемся к серверу и разворачиваем проект
echo "🔧 Разворачиваем проект на сервере..."
ssh ${SERVER_USER}@${SERVER_HOST} << 'EOF'
    # Останавливаем старый контейнер если есть
    if [ -d "/home/server/lovifyy_bot" ]; then
        cd /home/server/lovifyy_bot
        docker-compose down 2>/dev/null || true
    fi
    
    # Создаем директорию проекта
    mkdir -p /home/server/lovifyy_bot
    cd /home/server/lovifyy_bot
    
    # Распаковываем новую версию
    tar -xzf ~/lovifyy_bot.tar.gz
    rm ~/lovifyy_bot.tar.gz
    
    # Создаем необходимые директории
    mkdir -p data/chats data/diaries data/exercises data/logs data/notifications data/backups
    
    # Устанавливаем права доступа
    chmod +x deploy.sh
    
    echo "✅ Проект развернут в /home/server/lovifyy_bot"
EOF

# 4. Очищаем локальный архив
rm ${PROJECT_NAME}.tar.gz

echo "✅ Деплой завершен!"
echo "🔗 Подключитесь к серверу: ssh ${SERVER_USER}@${SERVER_HOST}"
echo "📁 Перейдите в директорию: cd ${REMOTE_PATH}"
echo "🚀 Запустите бота: docker-compose up -d"
