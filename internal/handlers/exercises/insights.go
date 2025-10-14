package exercises

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/godofphonk/lovifyy-bot/internal/ai"
	"github.com/godofphonk/lovifyy-bot/internal/history"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleInsightGenderChoice показывает выбор гендера для генерации инсайта как в legacy
func (h *Handler) HandleInsightGenderChoice(callbackQuery *tgbotapi.CallbackQuery, week int) error {
	response := fmt.Sprintf("🔍 Персональный инсайт (%d неделя)\n\n"+
		"Для кого вы хотите получить персональный инсайт?", week)

	// Создаем кнопки выбора гендера
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Для парня", fmt.Sprintf("insight_male_%d", week)),
			tgbotapi.NewInlineKeyboardButtonData("👩 Для девушки", fmt.Sprintf("insight_female_%d", week)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleInsightGender обрабатывает выбор гендера для инсайта - генерирует реальный AI инсайт
func (h *Handler) HandleInsightGender(callbackQuery *tgbotapi.CallbackQuery, data string, historyManager *history.Manager, aiClient *ai.OpenAIClient) error {
	// Парсим данные: insight_<gender>_<week>
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return fmt.Errorf("invalid insight callback data: %s", data)
	}

	gender := parts[1]
	weekStr := parts[2]
	weekNum, err := strconv.Atoi(weekStr)
	if err != nil {
		return fmt.Errorf("invalid week number: %s", weekStr)
	}

	userID := callbackQuery.From.ID

	var genderName string
	var genderEmoji string
	if gender == "male" {
		genderName = "парня"
		genderEmoji = "👨"
	} else {
		genderName = "девушки"
		genderEmoji = "👩"
	}

	// Отправляем сообщение о начале генерации
	processingMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
		fmt.Sprintf("🔍 Генерирую персональный инсайт для %s %s (неделя %d)...\n\n⏳ Анализирую записи в дневнике...", 
			genderEmoji, genderName, weekNum))
	_, err = h.bot.Send(processingMsg)
	if err != nil {
		return err
	}

	// Получаем записи дневника для конкретной недели и гендера (новый структурированный подход)
	weekEntries, err := historyManager.GetAllDiaryEntriesForWeekAndGender(userID, gender, weekNum)
	if err != nil || len(weekEntries) == 0 {
		response := fmt.Sprintf("🔍 Персональный инсайт для %s %s (неделя %d)\n\n"+
			"📝 Для создания качественного инсайта мне нужны ваши записи в дневнике для %d недели.\n\n"+
			"Пожалуйста, сначала сделайте записи в дневнике:\n"+
			"• Используйте кнопку \"📝 Мини дневник\"\n"+
			"• Выберите %s %s\n"+
			"• Выберите неделю %d\n"+
			"• Сделайте записи в разных категориях:\n"+
			"  - 💭 Личные мысли\n"+
			"  - ❓ Ответы на вопросы\n"+
			"  - 👫 Ответы на совместные вопросы\n\n"+
			"После этого вернитесь к инсайту для получения персонального анализа!", 
			genderEmoji, genderName, weekNum, weekNum, genderEmoji, genderName, weekNum)
		
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err = h.bot.Send(msg)
		return err
	}

	// Загружаем данные недели для контекста
	weekData, err := h.exerciseManager.GetWeekExercise(weekNum)
	if err != nil {
		return fmt.Errorf("failed to load week %d data: %v", weekNum, err)
	}

	// Формируем промпт для AI
	prompt := fmt.Sprintf(`Ты - опытный психолог по отношениям. Проанализируй записи в дневнике и создай персональный инсайт.

КОНТЕКСТ НЕДЕЛИ %d:
Тема: %s
Инсайт недели: %s

ЗАПИСИ В ДНЕВНИКЕ (%s):
`, weekNum, weekData.Title, weekData.Insights, genderName)

	for i, entry := range weekEntries {
		prompt += fmt.Sprintf("%d. [%s] %s: %s\n", i+1, entry.Timestamp.Format("02.01"), entry.Type, entry.Entry)
	}

	prompt += fmt.Sprintf(`
ЗАДАЧА:
Создай персональный инсайт для %s на основе записей в дневнике. Инсайт должен:

1. 🔍 АНАЛИЗ: Выдели ключевые темы и паттерны из записей
2. 💡 ИНСАЙТЫ: Дай 2-3 важных наблюдения о развитии отношений
3. 🎯 РЕКОМЕНДАЦИИ: Предложи конкретные шаги для дальнейшего роста
4. 🌟 МОТИВАЦИЯ: Отметь позитивные изменения и прогресс

Стиль: теплый, поддерживающий, профессиональный
Длина: 200-300 слов
Используй эмодзи для структуры`, genderName)

	// Генерируем инсайт с помощью AI
	if aiClient == nil {
		response := fmt.Sprintf("🔍 Персональный инсайт для %s %s (неделя %d)\n\n"+
			"❌ AI сервис временно недоступен. Попробуйте позже.\n\n"+
			"📊 Найдено записей в дневнике: %d", 
			genderEmoji, genderName, weekNum, len(weekEntries))
		
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err = h.bot.Send(msg)
		return err
	}

	insight, err := aiClient.Generate(prompt)
	if err != nil {
		response := fmt.Sprintf("🔍 Персональный инсайт для %s %s (неделя %d)\n\n"+
			"❌ Ошибка при генерации инсайта: %v\n\n"+
			"📊 Найдено записей в дневнике: %d\n"+
			"Попробуйте позже или обратитесь к администратору.", 
			genderEmoji, genderName, weekNum, err, len(weekEntries))
		
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
		_, err = h.bot.Send(msg)
		return err
	}

	// Отправляем готовый инсайт
	response := fmt.Sprintf("🔍 Персональный инсайт для %s %s (неделя %d)\n\n%s\n\n"+
		"📊 Проанализировано записей: %d\n"+
		"📅 Период анализа: неделя %d", 
		genderEmoji, genderName, weekNum, insight, len(weekEntries), weekNum)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, response)
	_, err = h.bot.Send(msg)
	return err
}
