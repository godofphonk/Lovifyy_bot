#!/bin/bash

# Запуск бота через OpenVPN
echo "🌐 Настраиваем OpenVPN для бота..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Убиваем старые OpenVPN процессы
sudo pkill openvpn 2>/dev/null || true

# Запускаем OpenVPN в фоне
echo "🚀 Запускаем OpenVPN..."
cd vpn_configs
sudo openvpn --config usa_tcp.ovpn --daemon --log /tmp/openvpn.log --writepid /tmp/openvpn.pid

# Ждем подключения
echo "⏳ Ждем подключения VPN..."
sleep 15

# Проверяем подключение
echo "🔍 Проверяем VPN подключение..."
if ip addr show | grep -q tun; then
    echo "✅ VPN подключен!"
    NEW_IP=$(curl -s --max-time 10 ifconfig.me || echo "неизвестен")
    echo "🌍 Новый IP: $NEW_IP"
else
    echo "❌ VPN не подключился"
    echo "📋 Логи OpenVPN:"
    tail -20 /tmp/openvpn.log 2>/dev/null || echo "Логи не найдены"
    exit 1
fi

# Возвращаемся в папку проекта
cd /home/server/lovifyy_bot

# Создаем docker-compose.yml для бота
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    network_mode: "host"
    env_file:
      - .env
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - SYSTEM_PROMPT=${SYSTEM_PROMPT}
      - ADMIN_IDS=${ADMIN_IDS}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - ./.env:/app/.env:ro
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Конфигурация создана"

# Запускаем бота
echo "🚀 Запускаем бота через VPN..."
docker-compose up -d

# Ждем запуска
sleep 10

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=15 lovifyy_bot

echo "✅ Бот запущен через OpenVPN!"
echo "🌍 IP адрес: $NEW_IP"
echo "🛑 Для остановки VPN: sudo pkill openvpn"
