#!/bin/bash

# Интерактивный скрипт развертывания Lovifyy Bot
# Запрашивает sudo пароль отдельно для безопасности

set -e

# Конфигурация подключения
SERVER_USER="server"
SERVER_HOST="192.168.0.104"
SERVER_PASSWORD="teec301210600644"
REMOTE_DIR="/opt/lovifyy_bot"
SERVICE_NAME="lovifyy-bot"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Развертывание Lovifyy Bot на сервере ${SERVER_HOST}${NC}"
echo ""

# Проверка наличия sshpass
if ! command -v sshpass &> /dev/null; then
    echo -e "${RED}❌ sshpass не найден. Установите его:${NC}"
    echo "Ubuntu/Debian: sudo apt-get install sshpass"
    echo "CentOS/RHEL: sudo yum install sshpass"
    echo "macOS: brew install hudochenkov/sshpass/sshpass"
    exit 1
fi

# Функция для выполнения команд на сервере
execute_remote() {
    sshpass -p "${SERVER_PASSWORD}" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_HOST} "$1"
}

# Функция для выполнения sudo команд на сервере
execute_remote_sudo() {
    echo -e "${YELLOW}🔐 Требуется пароль sudo для пользователя ${SERVER_USER}:${NC}"
    sshpass -p "${SERVER_PASSWORD}" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_HOST} "echo '${SERVER_SUDO_PASSWORD}' | sudo -S $1"
}

# Запрос sudo пароля
echo -e "${BLUE}Введите sudo пароль для пользователя ${SERVER_USER}:${NC}"
read -s SERVER_SUDO_PASSWORD
echo ""

# Проверка доступности сервера
echo -e "${YELLOW}📡 Проверяю доступность сервера...${NC}"
if ! execute_remote "echo 'Сервер доступен'" > /dev/null 2>&1; then
    echo -e "${RED}❌ Не удалось подключиться к серверу${NC}"
    exit 1
fi

# Проверка sudo пароля
echo -e "${YELLOW}🔐 Проверяю sudo пароль...${NC}"
if ! execute_remote_sudo "echo 'Sudo доступ разрешен'" > /dev/null 2>&1; then
    echo -e "${RED}❌ Неверный sudo пароль${NC}"
    exit 1
fi

# Шаг 1: Установка Docker и Docker Compose
echo -e "${YELLOW}📦 Устанавливаю Docker и Docker Compose...${NC}"
execute_remote_sudo "
# Обновление пакетов
apt-get update

# Установка зависимостей
apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

# Добавление официального GPG ключа Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Добавление репозитория Docker
echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# Установка Docker Engine
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io

# Установка Docker Compose
curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-\$(uname -s)-\$(uname -m)\" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Добавление пользователя в группу docker
usermod -aG docker \$USER

# Создание директории для бота
mkdir -p ${REMOTE_DIR}
chown \$USER:\$USER ${REMOTE_DIR}
"

# Шаг 2: Копирование файлов проекта
echo -e "${YELLOW}📋 Копирую файлы проекта на сервер...${NC}"
# Создаем временный архив
tar -czf /tmp/lovifyy_bot.tar.gz \
    --exclude='.git' \
    --exclude='build' \
    --exclude='coverage.out' \
    --exclude='coverage.html' \
    --exclude='.env' \
    --exclude='deploy*.sh' \
    .

# Копируем архив на сервер
sshpass -p "${SERVER_PASSWORD}" scp -o StrictHostKeyChecking=no /tmp/lovifyy_bot.tar.gz ${SERVER_USER}@${SERVER_HOST}:/tmp/

# Распаковываем на сервере
execute_remote "
cd ${REMOTE_DIR}
tar -xzf /tmp/lovifyy_bot.tar.gz
rm /tmp/lovifyy_bot.tar.gz
"

# Удаляем временный архив
rm /tmp/lovifyy_bot.tar.gz

# Шаг 3: Создание .env файла на сервере
echo -e "${YELLOW}⚙️ Создаю файл конфигурации .env...${NC}"
execute_remote "
cd ${REMOTE_DIR}
if [ ! -f .env ]; then
    cp .env.example .env
    echo 'Файл .env создан из шаблона'
fi
"

# Шаг 4: Отключение прокси если не нужен (закомментируем если нет SOCKS5 прокси)
echo -e "${YELLOW}🌐 Настраиваю сетевые параметры...${NC}"
execute_remote "
cd ${REMOTE_DIR}
# Проверяем, нужно ли использовать прокси
read -p 'Используется SOCKS5 прокси на сервере? (y/N): ' -n 1 -r
echo
if [[ ! \$REPLY =~ ^[Yy]\$ ]]; then
    echo 'Отключаю прокси в docker-compose.yml...'
    sed -i '/HTTP_PROXY/d' docker-compose.yml
    sed -i '/HTTPS_PROXY/d' docker-compose.yml
    echo 'Прокси отключен'
fi
"

# Шаг 5: Создание systemd сервиса для автозапуска
echo -e "${YELLOW}🔧 Настраиваю автозапуск...${NC}"
execute_remote_sudo "
tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null <<'EOF'
[Unit]
Description=Lovifyy Telegram Bot
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${REMOTE_DIR}
ExecStart=/usr/local/bin/docker-compose up -d
ExecStop=/usr/local/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ${SERVICE_NAME}
"

# Шаг 6: Запуск бота
echo -e "${YELLOW}🚀 Запускаю бота...${NC}"
execute_remote "
cd ${REMOTE_DIR}
docker-compose down 2>/dev/null || true
docker-compose up -d --build
"

# Шаг 7: Проверка статуса
echo -e "${YELLOW}🏥 Проверяю статус бота...${NC}"
execute_remote "
cd ${REMOTE_DIR}
echo '=== Статус контейнера ==='
docker-compose ps
echo ''
echo '=== Логи контейнера (последние 20 строк) ==='
docker-compose logs --tail=20
"

# Вывод инструкций
echo ""
echo -e "${GREEN}✅ Развертывание завершено!${NC}"
echo ""
echo -e "${YELLOW}⚠️  ВАЖНО: Перед запуском бота настройте .env файл:${NC}"
echo "1. Подключитесь к серверу: ssh ${SERVER_USER}@${SERVER_HOST}"
echo "2. Отредактируйте .env файл: nano ${REMOTE_DIR}/.env"
echo "3. Добавьте реальные токены:"
echo "   - TELEGRAM_BOT_TOKEN (от @BotFather)"
echo "   - OPENAI_API_KEY (с https://platform.openai.com/api-keys)"
echo "   - ADMIN_IDS (ID администраторов через запятую)"
echo "4. Перезапустите бота: cd ${REMOTE_DIR} && docker-compose restart"
echo ""
echo -e "${YELLOW}Полезные команды:${NC}"
echo "- Статус сервиса: sudo systemctl status ${SERVICE_NAME}"
echo "- Логи бота: cd ${REMOTE_DIR} && docker-compose logs -f"
echo "- Перезапуск: cd ${REMOTE_DIR} && docker-compose restart"
echo "- Остановка: cd ${REMOTE_DIR} && docker-compose down"
echo ""
echo -e "${GREEN}🎉 Бот развернут на сервере!${NC}"
