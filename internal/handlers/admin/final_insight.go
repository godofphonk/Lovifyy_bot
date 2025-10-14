package admin

import (
	"fmt"
	"strings"

	"github.com/godofphonk/lovifyy-bot/internal/ai"
	"github.com/godofphonk/lovifyy-bot/internal/history"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleFinalInsightMenu обрабатывает меню финального инсайта
func (h *Handler) HandleFinalInsightMenu(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID

	if !h.userManager.IsAdmin(userID) {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Эта функция доступна только администраторам.")
		_, err := h.bot.Send(msg)
		return err
	}

	// Отправляем сообщение о завершении месяца
	finalMessage := "🎉 Вы прошли целый месяц вместе со мной и сделали большой шаг в ваших отношениях. 💖\n\n" +
		"Каждый маленький шаг, каждая честная беседа и внимание друг к другу укрепляют вашу связь.\n\n" +
		"Горжусь вами! Продолжайте замечать друг друга, делиться чувствами и радоваться маленьким успехам. " +
		"Вы замечательная пара! 🫂🎀"

	// Создаем кнопку для получения финального инсайта
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Получить финальный инсайт", "generate_final_insight"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, finalMessage)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	return err
}

// HandleGenerateFinalInsight генерирует и отправляет финальный инсайт
func (h *Handler) HandleGenerateFinalInsight(callbackQuery *tgbotapi.CallbackQuery, historyManager *history.Manager, aiClient *ai.OpenAIClient) error {
	userID := callbackQuery.From.ID

	// Отправляем сообщение о начале генерации
	processingMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
		"⏳ Генерирую персональный финальный инсайт на основе вашей истории...")
	_, err := h.bot.Send(processingMsg)
	if err != nil {
		return err
	}

	// Получаем всю историю пользователя
	chatHistory, err := historyManager.GetUserHistory(userID, 0)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
			"❌ Ошибка при получении истории чатов")
		h.bot.Send(errorMsg)
		return err
	}

	diaryHistory, err := historyManager.GetUserDiary(userID, 0)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
			"❌ Ошибка при получении истории дневника")
		h.bot.Send(errorMsg)
		return err
	}

	// Проверяем, есть ли записи для мужчины и женщины
	hasMaleEntries := false
	hasFemaleEntries := false

	// Проверяем историю дневника на наличие гендерных записей
	for _, entry := range diaryHistory {
		lowerMsg := strings.ToLower(entry.Entry)
		
		// Ищем упоминания о парне/мужчине
		if strings.Contains(lowerMsg, "парень") || 
		   strings.Contains(lowerMsg, "мужчина") ||
		   strings.Contains(lowerMsg, "boyfriend") ||
		   strings.Contains(lowerMsg, "муж") {
			hasMaleEntries = true
		}
		
		// Ищем упоминания о девушке/женщине
		if strings.Contains(lowerMsg, "девушка") || 
		   strings.Contains(lowerMsg, "женщина") ||
		   strings.Contains(lowerMsg, "girlfriend") ||
		   strings.Contains(lowerMsg, "жена") {
			hasFemaleEntries = true
		}
	}

	// Также проверяем историю чатов
	for _, entry := range chatHistory {
		lowerMsg := strings.ToLower(entry.Message)
		
		if strings.Contains(lowerMsg, "парень") || 
		   strings.Contains(lowerMsg, "мужчина") ||
		   strings.Contains(lowerMsg, "boyfriend") ||
		   strings.Contains(lowerMsg, "муж") {
			hasMaleEntries = true
		}
		
		if strings.Contains(lowerMsg, "девушка") || 
		   strings.Contains(lowerMsg, "женщина") ||
		   strings.Contains(lowerMsg, "girlfriend") ||
		   strings.Contains(lowerMsg, "жена") {
			hasFemaleEntries = true
		}
	}

	// Формируем контекст для AI
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Вот полная история взаимодействий пользователя за месяц:\n\n")
	
	contextBuilder.WriteString("=== ИСТОРИЯ ЧАТОВ ===\n")
	for i, entry := range chatHistory {
		contextBuilder.WriteString(fmt.Sprintf("Неделя %d: %s\n", i+1, entry.Message))
	}
	
	contextBuilder.WriteString("\n=== ЗАПИСИ В ДНЕВНИКЕ ===\n")
	for i, entry := range diaryHistory {
		contextBuilder.WriteString(fmt.Sprintf("Запись %d: %s\n", i+1, entry.Entry))
	}

	// Создаем промпт для финального инсайта
	prompt := "Ты - эксперт по отношениям и психолог. На основе предоставленной истории создай финальный инсайт о развитии отношений пары за месяц.\n\n" +
		"ВАЖНО: Анализируй реальные данные из истории, не придумывай факты. Если данных мало, сосредоточься на том, что есть.\n\n" +
		"Структура ответа должна быть такой:\n" +
		"🌟 **В первую неделю** вы начинали с...\n" +
		"💭 **Во вторую неделю** вы задумывались о...\n" +
		"🚀 **На третьей неделе** вы сделали большой шаг и преодолели...\n" +
		"💖 **К четвертой неделе** вы достигли...\n\n" +
		"🎯 **Ваши главные достижения:**\n" +
		"- Перечисли конкретные успехи на основе истории\n\n" +
		"🌈 **Что дальше:**\n" +
		"- Дай рекомендации для продолжения развития отношений\n\n" +
		"Тон: теплый, поддерживающий, вдохновляющий. Покажи реальный прогресс и рост отношений на основе данных.\n\n" +
		contextBuilder.String()

	// Генерируем инсайт через AI
	insight, err := aiClient.Generate(prompt)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, 
			"❌ Ошибка при генерации инсайта. Попробуйте позже.")
		h.bot.Send(errorMsg)
		return err
	}

	// Если есть записи и для мужчины, и для женщины, генерируем персональные инсайты
	if hasMaleEntries && hasFemaleEntries {
		// Генерируем инсайт для девушки
		femalePrompt := prompt + "\n\nСосредоточься на перспективе и развитии ДЕВУШКИ в отношениях. Начни с 'Для девушки:'"
		femaleInsight, err := aiClient.Generate(femalePrompt)
		if err == nil {
			finalInsightMsg := "👩 **Для девушки:**\n\n" + femaleInsight
			msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, finalInsightMsg)
			msg.ParseMode = "Markdown"
			h.bot.Send(msg)
		}

		// Генерируем инсайт для парня
		malePrompt := prompt + "\n\nСосредоточься на перспективе и развитии ПАРНЯ в отношениях. Начни с 'Для парня:'"
		maleInsight, err := aiClient.Generate(malePrompt)
		if err == nil {
			finalInsightMsg := "👨 **Для парня:**\n\n" + maleInsight
			msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, finalInsightMsg)
			msg.ParseMode = "Markdown"
			h.bot.Send(msg)
		}
	} else {
		// Отправляем общий инсайт
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, insight)
		_, err = h.bot.Send(msg)
	}

	return err
}
