#!/bin/bash

# Скрипт автоматического развертывания Lovifyy Bot на домашнем сервере
# Требования: sshpass должен быть установлен локально

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
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Начинаю развертывание Lovifyy Bot на сервере ${SERVER_HOST}${NC}"

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

# Проверка доступности сервера
echo -e "${YELLOW}📡 Проверяю доступность сервера...${NC}"
if ! execute_remote "echo 'Сервер доступен'" > /dev/null 2>&1; then
    echo -e "${RED}❌ Не удалось подключиться к серверу${NC}"
    exit 1
fi

# Шаг 1: Установка Docker и Docker Compose
echo -e "${YELLOW}📦 Устанавливаю Docker и Docker Compose...${NC}"
execute_remote "
# Обновление пакетов
sudo apt-get update

# Установка зависимостей
sudo apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

# Добавление официального GPG ключа Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Добавление репозитория Docker
echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Установка Docker
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Установка Docker Compose
sudo curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-\$(uname -s)-\$(uname -m)\" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Добавление пользователя в группу docker
sudo usermod -aG docker \$USER

# Создание директории для бота
sudo mkdir -p ${REMOTE_DIR}
sudo chown \$USER:\$USER ${REMOTE_DIR}
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
    echo -e '${YELLOW}⚠️  ВНИМАНИЕ: Отредактируйте .env файл на сервере с реальными токенами!${NC}'
    echo 'Файл находится по пути: ${REMOTE_DIR}/.env'
    echo 'Необходимые переменные:'
    echo '- TELEGRAM_BOT_TOKEN (получить от @BotFather)'
    echo '- OPENAI_API_KEY (получить с https://platform.openai.com/api-keys)'
    echo '- ADMIN_IDS (ID администраторов бота через запятую)'
fi
"

# Шаг 4: Создание systemd сервиса для автозапуска
echo -e "${YELLOW}🔧 Настраиваю автозапуск...${NC}"
execute_remote "
sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null <<EOF
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

sudo systemctl daemon-reload
sudo systemctl enable ${SERVICE_NAME}
"

# Шаг 5: Запуск бота
echo -e "${YELLOW}🚀 Запускаю бота...${NC}"
execute_remote "
cd ${REMOTE_DIR}
docker-compose down
docker-compose up -d --build
"

# Шаг 6: Проверка статуса
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
echo -e "${GREEN}✅ Развертывание завершено!${NC}"
echo ""
echo -e "${YELLOW}Дальнейшие действия:${NC}"
echo "1. Подключитесь к серверу: ssh ${SERVER_USER}@${SERVER_HOST}"
echo "2. Отредактируйте .env файл: nano ${REMOTE_DIR}/.env"
echo "3. Перезапустите бота: cd ${REMOTE_DIR} && docker-compose restart"
echo ""
echo -e "${YELLOW}Полезные команды:${NC}"
echo "- Статус: sudo systemctl status ${SERVICE_NAME}"
echo "- Логи: cd ${REMOTE_DIR} && docker-compose logs -f"
echo "- Перезапуск: cd ${REMOTE_DIR} && docker-compose restart"
echo "- Остановка: cd ${REMOTE_DIR} && docker-compose down"
echo ""
echo -e "${GREEN}🎉 Бот готов к работе!${NC}"
