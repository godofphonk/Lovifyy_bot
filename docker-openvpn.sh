#!/bin/bash

# Запуск бота через OpenVPN в Docker
echo "🌐 Настраиваем OpenVPN через Docker..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Создаем docker-compose.yml с OpenVPN
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  # OpenVPN контейнер
  openvpn:
    image: dperson/openvpn-client
    container_name: openvpn_client
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun
    volumes:
      - ./vpn_configs:/vpn:ro
    environment:
      - VPN=usa_tcp.ovpn
    restart: unless-stopped
    dns:
      - 8.8.8.8
      - 8.8.4.4

  # Telegram бот через OpenVPN
  lovifyy_bot:
    build: .
    container_name: lovifyy_bot
    network_mode: "service:openvpn"
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
    depends_on:
      - openvpn
EOF

echo "✅ Конфигурация Docker OpenVPN создана"

# Запускаем контейнеры
echo "🚀 Запускаем OpenVPN и бота..."
docker-compose up -d

# Ждем запуска
sleep 20

# Проверяем статус
echo "📊 Статус контейнеров:"
docker-compose ps

# Проверяем логи OpenVPN
echo "📋 Логи OpenVPN:"
docker logs openvpn_client --tail=10

# Проверяем логи бота
echo "📋 Логи бота:"
docker logs lovifyy_bot --tail=10

# Проверяем IP через бота
echo "🌍 Проверяем IP через контейнер:"
docker exec lovifyy_bot curl -s --max-time 10 ifconfig.me 2>/dev/null || echo "Не удалось получить IP"

echo "✅ Настройка завершена!"
