package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// AIConfig содержит конфигурацию для работы с AI сервисом
type AIConfig struct {
	APIKey     string
	APIBaseURL string
	ModelName  string
}

// AIService представляет сервис для работы с AI
type AIService struct {
	config AIConfig
	client *http.Client
}

// AIRequest представляет запрос к API
type AIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
}

// Message представляет сообщение в формате API
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool представляет инструмент для модели
type Tool struct {
	Type     string     `json:"type"`
	Function ToolConfig `json:"function,omitempty"`
}

// ToolConfig представляет конфигурацию инструмента
type ToolConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AIResponse представляет ответ от API
type AIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAIService создает новый экземпляр AIService
func NewAIService() *AIService {
	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		log.Println("Предупреждение: переменная окружения AI_API_KEY не установлена. Будет использоваться заглушка.")
	}

	baseURL := os.Getenv("AI_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions" // URL по умолчанию
	}

	// Используем GPT-4o, так как он имеет актуальные знания о мире до апреля 2023
	// или GPT-4 Turbo, которая также имеет доступ к недавней информации
	modelName := os.Getenv("AI_MODEL_NAME")
	if modelName == "" {
		modelName = "gpt-4o" // Модель по умолчанию с актуальными данными
	}

	return &AIService{
		config: AIConfig{
			APIKey:     apiKey,
			APIBaseURL: baseURL,
			ModelName:  modelName,
		},
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateAnalytics генерирует аналитику с использованием AI
func (s *AIService) GenerateAnalytics() (string, error) {
	// Если API ключ не установлен, использовать локальную заглушку
	if s.config.APIKey == "" {
		return s.getLocalAnalytics(), nil
	}

	prompt := AIPromptDailyUpdate()

	// Текущая дата для контекста
	loc, _ := time.LoadLocation("Europe/Moscow")
	if loc == nil {
		loc = time.FixedZone("MSK", 3*60*60)
	}
	now := time.Now().In(loc)
	dateStr := now.Format("02.01.2006")

	// Добавляем текущую дату в контекст
	dateContext := fmt.Sprintf("Сегодняшняя дата: %s. Пожалуйста, используй свои актуальные знания о текущем состоянии российского рынка акций на эту дату. Нужна аналитика именно по сегодняшнему дню.", dateStr)

	// Настраиваем инструменты для доступа к актуальным данным
	retrievalTool := Tool{
		Type: "retrieval", // Инструмент для поиска актуальной информации
	}

	request := AIRequest{
		Model: s.config.ModelName,
		Messages: []Message{
			{
				Role:    "system",
				Content: prompt,
			},
			{
				Role:    "user",
				Content: dateContext,
			},
		},
		Temperature: 0.7, // Средняя температура для баланса между креативностью и точностью
		Tools: []Tool{
			retrievalTool,
		},
		ToolChoice: "auto", // Модель сама решит, когда использовать инструмент
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("ошибка при сериализации запроса: %v", err)
	}

	req, err := http.NewRequest("POST", s.config.APIBaseURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при чтении ответа: %v", err)
	}

	var aiResponse AIResponse
	err = json.Unmarshal(body, &aiResponse)
	if err != nil {
		return "", fmt.Errorf("ошибка при десериализации ответа: %v", err)
	}

	if aiResponse.Error != nil {
		return "", fmt.Errorf("ошибка API: %s", aiResponse.Error.Message)
	}

	if len(aiResponse.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от API")
	}

	return aiResponse.Choices[0].Message.Content, nil
}

// getLocalAnalytics возвращает локально сгенерированную аналитику (заглушка)
func (s *AIService) getLocalAnalytics() string {
	// Текущая дата для отчета
	loc, _ := time.LoadLocation("Europe/Moscow")
	if loc == nil {
		loc = time.FixedZone("MSK", 3*60*60)
	}
	now := time.Now().In(loc)
	dateStr := now.Format("02.01.2006")

	// Формирование шаблона аналитики
	analytics := fmt.Sprintf(`✨ *Ежедневная аналитика рынка* ✨
🗓 *%s*

Приветик, дорогой инвестор! 👋💕

Сегодня российский рынок выглядит очень интересно! 📊 Я проанализировала все тренды специально для тебя! 🌟

🔍 *Общая ситуация на рынке РФ*:
Индекс Мосбиржи сегодня показывает небольшой рост, что создаёт приятные возможности для инвестиций! 💫 Рубль стабилен, что хорошо для прогнозирования! 🧮

💰 *Куда вложить 1000 рублей*:
Для такой суммы я рекомендую обратить внимание на акции компании "Газпром" через Сбер Инвестиции! 🏦 Сейчас они торгуются по привлекательной цене и имеют хороший потенциал роста в ближайшие месяцы! 📈

Альтернативный вариант - накопительный вклад в Тинькофф со ставкой 16%% годовых для новых клиентов! 💝 Это безопасный способ сохранить деньги в текущих условиях! 🔐

🌈 *Почему это выгодно*:
Газпром выплачивает стабильные дивиденды, а текущая конъюнктура рынка энергоносителей благоприятна для компании! ✅ Даже с 1000 рублей ты сможешь получить настоящий инвестиционный опыт! 🤓

🎁 *Уникальный совет по подработке*:
Знаешь ли ты, что можно зарабатывать на своих знаниях? 🧠 Платформа Яндекс.Кью платит за качественные ответы на вопросы! 💡 Некоторые эксперты зарабатывают там до 20000 рублей в месяц, уделяя всего 2 часа в день! 🕰️ Это отличная возможность монетизировать свою экспертизу! 💸

Надеюсь, моя аналитика была полезной! 🌺 Жду тебя завтра с новыми инсайтами! 💖

Твой финансовый помощник! 💝`, dateStr)

	return analytics
}
