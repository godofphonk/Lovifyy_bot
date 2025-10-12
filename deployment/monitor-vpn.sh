#!/bin/bash

# Мониторинг Planet VPN и автоперезапуск
echo "🔍 Проверяем статус Planet VPN..."

# Проверяем процесс sing-box
if ! pgrep -f "sing-box" > /dev/null; then
    echo "❌ sing-box не запущен! Перезапускаем..."
    cd /home/server/lovifyy_bot
    ./planet-vpn.sh
    exit 1
fi

# Проверяем работу прокси
VPN_IP=$(curl -x socks5://127.0.0.1:1080 --max-time 10 -s ifconfig.me 2>/dev/null)
if [ -z "$VPN_IP" ] || [ "$VPN_IP" = "192.168.0.102" ]; then
    echo "❌ VPN не работает! IP: $VPN_IP"
    echo "🔄 Перезапускаем sing-box..."
    pkill sing-box
    sleep 5
    cd /home/server/lovifyy_bot
    /home/server/sing-box-1.12.9-linux-amd64/sing-box run -c sing-box-config.json > /tmp/sing-box.log 2>&1 &
    sleep 10
    
    # Проверяем снова
    NEW_IP=$(curl -x socks5://127.0.0.1:1080 --max-time 10 -s ifconfig.me 2>/dev/null)
    if [ -n "$NEW_IP" ] && [ "$NEW_IP" != "192.168.0.102" ]; then
        echo "✅ VPN восстановлен! IP: $NEW_IP"
    else
        echo "❌ Не удалось восстановить VPN"
        exit 1
    fi
else
    echo "✅ VPN работает! IP: $VPN_IP"
fi

# Проверяем бота
if ! docker ps | grep -q "lovifyy_bot.*Up"; then
    echo "❌ Бот не запущен! Перезапускаем..."
    cd /home/server/lovifyy_bot
    docker-compose up -d
fi

echo "✅ Все системы работают!"
