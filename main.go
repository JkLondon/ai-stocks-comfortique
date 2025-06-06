package main

import (
	"log"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// Время отправки ежедневной аналитики (Московское время, UTC+3)
const (
	DAILY_HOUR    = 10 // 10 утра по Москве
	DAILY_MINUTE  = 0
	ADMIN_USER_ID = 449066543 // ID администратора бота
)

func main() {
	// Загрузка переменных окружения из .env файла
	err := godotenv.Load()
	if err != nil {
		log.Println("Предупреждение: Не удалось загрузить файл .env:", err)
		log.Println("Продолжаем с переменными окружения системы...")
	} else {
		log.Println("Переменные окружения успешно загружены из .env файла")
	}

	// Получение токена из переменной окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Необходимо установить переменную окружения TELEGRAM_BOT_TOKEN")
	}

	// Получение API ключа и имени модели из переменных окружения
	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		log.Fatal("Необходимо установить переменную окружения AI_API_KEY")
	}

	modelName := os.Getenv("AI_MODEL_NAME")
	if modelName == "" {
		modelName = "gpt-4o" // Значение по умолчанию, если переменная окружения не установлена
		log.Println("Переменная AI_MODEL_NAME не установлена, используется модель по умолчанию:", modelName)
	}

	// Инициализация бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Бот авторизован как %s", bot.Self.UserName)

	// Создаем AI сервис с передачей необходимых параметров
	aiService := NewAIService(apiKey, modelName)

	// Настройка получения обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Канал для запуска ежедневной аналитики
	dailyTicker := scheduleDaily(DAILY_HOUR, DAILY_MINUTE)

	// Список подписанных чатов (в реальном проекте лучше использовать базу данных)
	subscribedChats := make(map[int64]bool)

	// Основной цикл обработки сообщений
	for {
		select {
		case update := <-updates:
			// Обработка сообщений от пользователей
			if update.Message != nil {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
				handleMessage(bot, update.Message, subscribedChats, aiService)
			}
		case <-dailyTicker:
			// Отправка ежедневной аналитики всем подписчикам
			sendDailyAnalytics(bot, subscribedChats, aiService)
		}
	}
}

// Обработка сообщений от пользователей
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, subscribedChats map[int64]bool, aiService *AIService) {
	chatID := message.Chat.ID
	userID := message.From.ID

	// Проверка, является ли пользователь администратором
	isAdmin := userID == ADMIN_USER_ID

	// Проверяем, что сообщение является командой (начинается с /)
	if len(message.Text) == 0 || message.Text[0] != '/' {
		// Если это не команда, игнорируем сообщение
		return
	}

	// Для команд, доступных всем пользователям
	switch message.Command() {
	case "start":
		// Приветственное сообщение
		welcomeText := `Привет! 👋 Я твой милый помощник по инвестициям! 💖

Я буду каждый день в 10:00 по Москве отправлять тебе аналитику по российскому рынку с рекомендациями куда вложить 1000 рублей! 💰

Используй команды:
/subscribe - подписаться на ежедневную аналитику 📊
/unsubscribe - отписаться от ежедневной аналитики 🚫
/analytics - получить аналитику прямо сейчас ✨`

		if isAdmin {
			welcomeText += "\n\n🔐 Вы администратор бота и имеете доступ ко всем функциям!"
		}

		msg := tgbotapi.NewMessage(chatID, welcomeText)
		bot.Send(msg)
		return
	case "subscribe", "unsubscribe", "analytics":
		// Проверяем, является ли пользователь админом для этих команд
		if !isAdmin {
			msg := tgbotapi.NewMessage(chatID, "Извините, но эта команда доступна только администратору бота! 🔒")
			bot.Send(msg)
			return
		}
	default:
		return
	}

	// Обработка команд для администратора
	switch message.Command() {
	case "subscribe":
		// Подписка на ежедневную аналитику
		subscribedChats[chatID] = true
		msg := tgbotapi.NewMessage(chatID, "Вы успешно подписались на ежедневную аналитику! 🎉 Ожидайте первый выпуск в 10:00 по Москве! 💖")
		bot.Send(msg)

	case "unsubscribe":
		// Отписка от ежедневной аналитики
		delete(subscribedChats, chatID)
		msg := tgbotapi.NewMessage(chatID, "Вы отписались от ежедневной аналитики 😢 Будем скучать! 💔")
		bot.Send(msg)

	case "analytics":
		// Отправка аналитики по запросу
		msg := tgbotapi.NewMessage(chatID, "Генерирую аналитику, пожалуйста, подождите... ⏳")
		sentMsg, _ := bot.Send(msg)

		analytics, err := aiService.GenerateAnalytics()
		if err != nil {
			log.Printf("Ошибка генерации аналитики: %v", err)
			errorMsg := tgbotapi.NewMessage(chatID, "Извини, произошла ошибка при генерации аналитики 😢 Попробуй позже! 💕")
			bot.Send(errorMsg)
			return
		}

		// Редактируем сообщение, заменяя его на аналитику
		editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, analytics)
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)
	}
}

// Планировщик ежедневных задач
func scheduleDaily(hour, minute int) <-chan time.Time {
	ticker := make(chan time.Time)

	go func() {
		for {
			// Получаем текущее время в Москве (UTC+3)
			loc, _ := time.LoadLocation("Europe/Moscow")
			if loc == nil {
				// Если не удалось загрузить локацию, используем UTC+3 вручную
				loc = time.FixedZone("MSK", 3*60*60)
			}
			now := time.Now().In(loc)

			// Вычисляем следующее время запуска
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
			if now.After(nextRun) {
				// Если текущее время после запланированного - переходим на следующий день
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// Ожидание до следующего запуска
			waitDuration := nextRun.Sub(now)
			log.Printf("Следующая отправка аналитики через %s", waitDuration)

			timer := time.NewTimer(waitDuration)
			<-timer.C

			// Отправляем сигнал в канал
			ticker <- time.Now()
		}
	}()

	return ticker
}

// Отправка ежедневной аналитики всем подписчикам
func sendDailyAnalytics(bot *tgbotapi.BotAPI, subscribedChats map[int64]bool, aiService *AIService) {
	log.Printf("Отправка ежедневной аналитики %d подписчикам", len(subscribedChats))

	analytics, err := aiService.GenerateAnalytics()
	if err != nil {
		log.Printf("Ошибка генерации ежедневной аналитики: %v", err)
		return
	}

	for chatID := range subscribedChats {
		msg := tgbotapi.NewMessage(chatID, analytics)
		msg.ParseMode = "Markdown"
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки аналитики пользователю %d: %v", chatID, err)
		}
	}
}
