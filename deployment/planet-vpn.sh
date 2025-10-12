#!/bin/bash

# Запуск бота через Planet VPN (VLESS)
echo "🌍 Настраиваем Planet VPN (VLESS) для бота..."

cd /home/server/lovifyy_bot

# Останавливаем текущие контейнеры
docker-compose down 2>/dev/null || true

# Создаем правильную конфигурацию sing-box
cat > sing-box-config.json << 'EOF'
{
  "log": {
    "level": "info"
  },
  "inbounds": [
    {
      "type": "mixed",
      "listen": "127.0.0.1",
      "listen_port": 1080,
      "sniff": true,
      "sniff_override_destination": true
    }
  ],
  "outbounds": [
    {
      "type": "vless",
      "tag": "planet-vpn",
      "server": "51.159.199.39",
      "server_port": 443,
      "uuid": "L85TPBRGBwPNzjk3tDd6ezf6J8iKB8",
      "flow": "xtls-rprx-vision",
      "tls": {
        "enabled": true,
        "server_name": "t.me",
        "utls": {
          "enabled": true,
          "fingerprint": "chrome"
        },
        "reality": {
          "enabled": true,
          "public_key": "mCb1gzQ26IuSBqMELd4plHBtpieED_ywh0PvO8P1VmA",
          "short_id": "01"
        }
      }
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "rules": [
      {
        "domain_suffix": [".ru", ".рф"],
        "outbound": "direct"
      }
    ],
    "final": "planet-vpn"
  }
}
EOF

echo "✅ Конфигурация создана!"

# Запускаем sing-box в фоне
echo "🚀 Запускаем Planet VPN..."
/home/server/sing-box-1.12.9-linux-amd64/sing-box run -c sing-box-config.json > /tmp/sing-box.log 2>&1 &
SING_PID=$!
echo "PID: $SING_PID"

# Ждем запуска
sleep 10

# Проверяем подключение
echo "🔍 Проверяем VPN подключение..."
if ps -p $SING_PID > /dev/null; then
    echo "✅ sing-box запущен!"
    
    # Тестируем прокси
    NEW_IP=$(curl -x socks5://127.0.0.1:1080 --max-time 15 -s ifconfig.me 2>/dev/null || echo "неизвестен")
    if [ "$NEW_IP" != "неизвестен" ]; then
        echo "🌍 Новый IP через VPN: $NEW_IP"
        echo "✅ Planet VPN работает!"
    else
        echo "❌ VPN не работает"
        echo "📋 Логи sing-box:"
        tail -10 /tmp/sing-box.log
        kill $SING_PID
        exit 1
    fi
else
    echo "❌ sing-box не запустился"
    echo "📋 Логи sing-box:"
    cat /tmp/sing-box.log
    exit 1
fi

# Создаем docker-compose.yml с прокси
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
      - HTTP_PROXY=socks5://127.0.0.1:1080
      - HTTPS_PROXY=socks5://127.0.0.1:1080
      - ALL_PROXY=socks5://127.0.0.1:1080
    volumes:
      - ./.env:/app/.env:ro
      - ./chat_history:/app/chat_history
      - ./diary_entries:/app/diary_entries
      - ./exercises:/app/exercises
    restart: unless-stopped
EOF

echo "✅ Docker конфигурация создана"

# Запускаем бота
echo "🚀 Запускаем бота через Planet VPN..."
docker-compose up -d

# Ждем запуска
sleep 15

# Проверяем статус
echo "📊 Статус контейнера:"
docker-compose ps

# Показываем логи
echo "📋 Логи бота:"
docker-compose logs --tail=20 lovifyy_bot

echo ""
echo "🎯 Planet VPN настроен!"
echo "🌍 IP адрес: $NEW_IP"
echo "🛑 Для остановки VPN: pkill sing-box"
echo "💾 PID sing-box: $SING_PID"
