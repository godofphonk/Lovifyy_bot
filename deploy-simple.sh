#!/bin/bash

# Упрощенный скрипт развертывания без требований sudo
# Использует Docker в пользовательской директории

set -e

# Конфигурация подключения
SERVER_USER="server"
SERVER_HOST="192.168.0.104"
SERVER_PASSWORD="teec301210600644"
REMOTE_DIR="$HOME/lovifyy_bot"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Простое развертывание Lovifyy Bot (без sudo)${NC}"

# Проверка наличия sshpass
if ! command -v sshpass &> /dev/null; then
    echo -e "${RED}❌ Установите sshpass: sudo apt-get install sshpass${NC}"
    exit 1
fi

# Функция для выполнения команд на сервере
execute_remote() {
    sshpass -p "${SERVER_PASSWORD}" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_HOST} "$1"
}

# Проверка доступности сервера
echo -e "${YELLOW}📡 Подключаюсь к серверу...${NC}"
if ! execute_remote "echo 'Сервер доступен'" > /dev/null 2>&1; then
    echo -e "${RED}❌ Не удалось подключиться${NC}"
    exit 1
fi

# Проверка наличия Docker
echo -e "${YELLOW}🐳 Проверяю Docker...${NC}"
if ! execute_remote "docker --version" > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker не установлен. Установите Docker вручную:${NC}"
    echo "1. Подключитесь: ssh ${SERVER_USER}@${SERVER_HOST}"
    echo "2. Выполните: curl -fsSL https://get.docker.com | sh"
    echo "3. Добавьте пользователя в группу: sudo usermod -aG docker \$USER"
    echo "4. Перезайдите: exit && ssh ${SERVER_USER}@${SERVER_HOST}"
    exit 1
fi

# Копирование проекта
echo -e "${YELLOW}📋 Копирую файлы проекта...${NC}"
tar -czf /tmp/lovifyy_bot.tar.gz \
    --exclude='.git' \
    --exclude='build' \
    --exclude='coverage.*' \
    --exclude='.env' \
    --exclude='deploy*.sh' \
    .

sshpass -p "${SERVER_PASSWORD}" scp -o StrictHostKeyChecking=no /tmp/lovifyy_bot.tar.gz ${SERVER_USER}@${SERVER_HOST}:/tmp/
execute_remote "mkdir -p ${REMOTE_DIR} && cd ${REMOTE_DIR} && tar -xzf /tmp/lovifyy_bot.tar.gz && rm /tmp/lovifyy_bot.tar.gz"
rm /tmp/lovifyy_bot.tar.gz

# Создание .env
echo -e "${YELLOW}⚙️ Создаю .env файл...${NC}"
execute_remote "
cd ${REMOTE_DIR}
if [ ! -f .env ]; then
    cp .env.example .env
    echo '✅ .env файл создан'
fi
"

# Настройка docker-compose без прокси
echo -e "${YELLOW}🌐 Настраиваю docker-compose...${NC}"
execute_remote "
cd ${REMOTE_DIR}
# Создаем упрощенный docker-compose без прокси
cat > docker-compose.simple.yml <<'EOF'
version: '3.8'

services:
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    environment:
      - TELEGRAM_BOT_TOKEN=\${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=\${SYSTEM_PROMPT}
      - ADMIN_IDS=\${ADMIN_IDS}
      - OPENAI_API_KEY=\${OPENAI_API_KEY}
      - ENABLE_PROMETHEUS=true
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    ports:
      - \"8080:8080\"
    healthcheck:
      test: [\"CMD\", \"pgrep\", \"-f\", \"./main\"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.5'
EOF
echo '✅ docker-compose.simple.yml создан'
"

# Запуск бота
echo -e "${YELLOW}🚀 Запускаю бота...${NC}"
execute_remote "
cd ${REMOTE_DIR}
docker-compose -f docker-compose.simple.yml down 2>/dev/null || true
docker-compose -f docker-compose.simple.yml up -d --build
sleep 5
"

# Проверка статуса
echo -e "${YELLOW}🏥 Проверяю статус...${NC}"
execute_remote "
cd ${REMOTE_DIR}
echo '=== Контейнер ==='
docker-compose -f docker-compose.simple.yml ps
echo ''
echo '=== Логи ==='
docker-compose -f docker-compose.simple.yml logs --tail=10
"

# Инструкции
echo ""
echo -e "${GREEN}✅ Бот развернут!${NC}"
echo ""
echo -e "${YELLOW}Следующие шаги:${NC}"
echo "1. SSH: ssh ${SERVER_USER}@${SERVER_HOST}"
echo "2. Редактировать .env: nano ${REMOTE_DIR}/.env"
echo "3. Перезапустить: cd ${REMOTE_DIR} && docker-compose -f docker-compose.simple.yml restart"
echo ""
echo -e "${YELLOW}Команды управления:${NC}"
echo "- Логи: cd ${REMOTE_DIR} && docker-compose -f docker-compose.simple.yml logs -f"
echo "- Стоп: cd ${REMOTE_DIR} && docker-compose -f docker-compose.simple.yml down"
echo "- Рестарт: cd ${REMOTE_DIR} && docker-compose -f docker-compose.simple.yml restart"
