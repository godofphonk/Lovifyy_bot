#!/bin/bash

# Финальный скрипт развертывания Lovifyy Bot
# Работает полностью в домашней директории пользователя без sudo

set -e

# Конфигурация
SERVER_USER="server"
SERVER_HOST="192.168.0.104"
SERVER_PASSWORD="teec301210600644"
REMOTE_DIR="/home/server/lovifyy_bot"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Развертывание Lovifyy Bot на ${SERVER_HOST}${NC}"

# Проверка sshpass
if ! command -v sshpass &> /dev/null; then
    echo -e "${RED}❌ Установите sshpass: sudo apt-get install sshpass${NC}"
    exit 1
fi

# Функция выполнения на сервере
execute_remote() {
    sshpass -p "${SERVER_PASSWORD}" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_HOST} "$1"
}

# Тест подключения
echo -e "${YELLOW}📡 Проверяю соединение...${NC}"
if ! execute_remote "echo 'OK'" > /dev/null 2>&1; then
    echo -e "${RED}❌ Нет подключения к серверу${NC}"
    exit 1
fi

# Подготовка директории
echo -e "${YELLOW}📁 Подготавливаю директорию...${NC}"
execute_remote "mkdir -p ${REMOTE_DIR} && chmod 755 ${REMOTE_DIR}"

# Копирование файлов
echo -e "${YELLOW}📋 Копирую проект...${NC}"
tar -czf /tmp/lovifyy.tar.gz \
    --exclude='.git' \
    --exclude='build' \
    --exclude='coverage.*' \
    --exclude='.env' \
    --exclude='deploy*.sh' \
    .

sshpass -p "${SERVER_PASSWORD}" scp -o StrictHostKeyChecking=no /tmp/lovifyy.tar.gz ${SERVER_USER}@${SERVER_HOST}:/tmp/
execute_remote "cd ${REMOTE_DIR} && tar -xzf /tmp/lovifyy.tar.gz && rm /tmp/lovifyy.tar.gz"
rm /tmp/lovifyy.tar.gz

# Создание .env
echo -e "${YELLOW}⚙️ Создаю .env...${NC}"
execute_remote "
cd ${REMOTE_DIR}
cp .env.example .env 2>/dev/null || echo 'TELEGRAM_BOT_TOKEN=your_token' > .env
echo 'OPENAI_API_KEY=your_key' >> .env
echo 'ADMIN_IDS=your_admin_id' >> .env
chmod 600 .env
"

# Создание docker-compose без прокси и без host сети
echo -e "${YELLOW}🐳 Создаю docker-compose.yml...${NC}"
execute_remote "
cd ${REMOTE_DIR}
cat > docker-compose.yml <<'EOF'
version: '3.8'

services:
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - ADMIN_IDS=${ADMIN_IDS}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - SYSTEM_PROMPT=${SYSTEM_PROMPT}
      - ENABLE_PROMETHEUS=false
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    ports:
      - '8080:8080'
    healthcheck:
      test: ['CMD', 'pgrep', '-f', './main']
      interval: 30s
      timeout: 10s
      retries: 3
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: '0.5'
EOF
"

# Создание скрипта запуска
echo -e "${YELLOW}📜 Создаю скрипты управления...${NC}"
execute_remote "
cd ${REMOTE_DIR}
cat > start.sh <<'EOF'
#!/bin/bash
cd $(dirname $0)
docker-compose down 2>/dev/null || true
docker-compose up -d --build
echo 'Бот запущен. Логи: docker-compose logs -f'
EOF

cat > stop.sh <<'EOF'
#!/bin/bash
cd $(dirname $0)
docker-compose down
echo 'Бот остановлен'
EOF

cat > logs.sh <<'EOF'
#!/bin/bash
cd $(dirname $0)
docker-compose logs -f
EOF

cat > status.sh <<'EOF'
#!/bin/bash
cd $(dirname $0)
echo '=== Статус контейнера ==='
docker-compose ps
echo ''
echo '=== Последние логи ==='
docker-compose logs --tail=20
EOF

chmod +x *.sh
"

# Запуск
echo -e "${YELLOW}🚀 Запускаю бота...${NC}"
execute_remote "
cd ${REMOTE_DIR}
./start.sh
sleep 3
./status.sh
"

# Инструкции
echo ""
echo -e "${GREEN}✅ Бот успешно развернут!${NC}"
echo ""
echo -e "${YELLOW}❗️ ВАЖНО - настройте .env файл:${NC}"
echo "1. Подключитесь: ssh ${SERVER_USER}@${SERVER_HOST}"
echo "2. Откройте .env: nano ${REMOTE_DIR}/.env"
echo "3. Вставьте реальные токены:"
echo "   - TELEGRAM_BOT_TOKEN (от @BotFather)"
echo "   - OPENAI_API_KEY (с openai.com)"
echo "   - ADMIN_IDS (ваш Telegram ID)"
echo "4. Перезапустите: cd ${REMOTE_DIR} && ./start.sh"
echo ""
echo -e "${YELLOW}📋 Команды управления:${NC}"
echo "- Статус: cd ${REMOTE_DIR} && ./status.sh"
echo "- Логи: cd ${REMOTE_DIR} && ./logs.sh"
echo "- Стоп: cd ${REMOTE_DIR} && ./stop.sh"
echo "- Запуск: cd ${REMOTE_DIR} && ./start.sh"
echo ""
echo -e "${GREEN}🎉 Готово! Бот работает на ${SERVER_HOST}:8080${NC}"
