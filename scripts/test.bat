@echo off
echo 🧪 Тестирование Ollama...
echo.

REM Переходим в корневую папку проекта
cd /d "%~dp0\.."

REM Запускаем тест
echo 🔍 Запускаем тест подключения...
go run -tags=test tests/ollama_test.go

pause
