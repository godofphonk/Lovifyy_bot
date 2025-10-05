@echo off
echo 🚀 Запуск Lovifyy Bot...
echo.

REM Проверяем что Ollama запущен
echo 🔍 Проверяем Ollama...
curl -s http://localhost:11434/api/tags > nul
if %errorlevel% neq 0 (
    echo ❌ Ollama не запущен! Запустите Ollama и попробуйте снова.
    pause
    exit /b 1
)
echo ✅ Ollama работает!

REM Переходим в корневую папку проекта
cd /d "%~dp0\.."

REM Запускаем бота
echo 🤖 Запускаем бота...
go run cmd/main.go

pause
