# 🚀 Настройка сервера для Lovifyy Bot

## Требования к серверу
- Ubuntu/Debian Linux
- Docker и Docker Compose
- SSH доступ
- Минимум 1GB RAM, 10GB диск

## 1. Подготовка сервера

### Подключение к серверу:
```bash
ssh server@192.168.0.102
```

### Установка Docker (если не установлен):
```bash
# Обновляем систему
sudo apt update && sudo apt upgrade -y

# Устанавливаем Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Добавляем пользователя в группу docker
sudo usermod -aG docker $USER

# Устанавливаем Docker Compose
sudo apt install docker-compose -y

# Перезагружаемся для применения изменений
sudo reboot
```

## 2. Деплой бота

### С локального компьютера:
```bash
# Делаем скрипт исполняемым
chmod +x deploy.sh

# Запускаем деплой
./deploy.sh
```

### На сервере после деплоя:
```bash
# Переходим в директорию проекта
cd /home/server/lovifyy_bot

# Копируем и настраиваем .env файл
cp .env.server .env
nano .env  # Заполняем токены и ID

# Запускаем бота
docker-compose up -d

# Проверяем статус
docker-compose ps
docker-compose logs -f lovifyy_bot
```

## 3. Управление ботом

### Основные команды:
```bash
# Запуск
docker-compose up -d

# Остановка
docker-compose down

# Перезапуск
docker-compose restart

# Логи
docker-compose logs -f lovifyy_bot

# Обновление (после нового деплоя)
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Мониторинг:
```bash
# Статус контейнеров
docker-compose ps

# Использование ресурсов
docker stats

# Размер образов
docker images
```

## 4. Автозапуск при перезагрузке

### Создаем systemd сервис:
```bash
sudo nano /etc/systemd/system/lovifyy-bot.service
```

### Содержимое сервиса:
```ini
[Unit]
Description=Lovifyy Bot
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/home/server/lovifyy_bot
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

### Активируем сервис:
```bash
sudo systemctl enable lovifyy-bot.service
sudo systemctl start lovifyy-bot.service
```

## 5. Резервное копирование

### Создаем скрипт бэкапа:
```bash
nano ~/backup_bot.sh
```

### Содержимое скрипта:
```bash
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/home/server/backups"
BOT_DIR="/home/server/lovifyy_bot"

mkdir -p $BACKUP_DIR

# Создаем архив данных
tar -czf $BACKUP_DIR/lovifyy_bot_$DATE.tar.gz \
    -C $BOT_DIR \
    chat_history \
    diary_entries \
    exercises \
    scheduled_notifications.json \
    .env

echo "Бэкап создан: $BACKUP_DIR/lovifyy_bot_$DATE.tar.gz"

# Удаляем старые бэкапы (старше 7 дней)
find $BACKUP_DIR -name "lovifyy_bot_*.tar.gz" -mtime +7 -delete
```

### Автоматический бэкап (cron):
```bash
chmod +x ~/backup_bot.sh
crontab -e

# Добавляем строку для ежедневного бэкапа в 3:00
0 3 * * * /home/server/backup_bot.sh
```

## 6. Безопасность

### Настройка файрвола:
```bash
# Разрешаем только SSH
sudo ufw allow ssh
sudo ufw enable

# Если нужен веб-интерфейс (опционально)
# sudo ufw allow 8080
```

### Обновление системы:
```bash
# Регулярно обновляем систему
sudo apt update && sudo apt upgrade -y

# Настраиваем автообновления безопасности
sudo apt install unattended-upgrades -y
sudo dpkg-reconfigure unattended-upgrades
```
