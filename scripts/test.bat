@echo off
echo 🧪 Тестирование Ollama...
echo.

REM Переходим в корневую папку проекта
cd /d "%~dp0\.."

REM Запускаем тесты
echo 🔍 Запускаем тесты подключения...
go test ./tests/ -v

pause
