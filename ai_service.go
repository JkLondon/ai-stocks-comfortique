package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// AIService взаимодействует с моделью AI
type AIService struct {
	apiKey            string
	apiURL            string
	modelName         string
	marketDataService *MarketDataService
}

// NewAIService создает новый экземпляр AIService
func NewAIService(apiKey, modelName string) *AIService {
	return &AIService{
		apiKey:            apiKey,
		apiURL:            "https://api.openai.com/v1/chat/completions",
		modelName:         modelName,
		marketDataService: NewMarketDataService(),
	}
}

// AIRequest представляет собой запрос к API
type AIRequest struct {
	Model       string      `json:"model"`
	Messages    []AIMessage `json:"messages"`
	Temperature float64     `json:"temperature"`
}

// AIMessage представляет собой сообщение в запросе к API
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIResponse представляет собой ответ от API
type AIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int       `json:"index"`
		Message      AIMessage `json:"message"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// GenerateAnalytics генерирует аналитику на основе текущего состояния рынка
func (s *AIService) GenerateAnalytics() (string, error) {
	// Получаем данные о рынке
	marketData, err := s.marketDataService.GetMarketData()
	if err != nil {
		log.Printf("Ошибка при получении данных о рынке: %v", err)
		// Продолжаем без данных о рынке
	}

	// Формируем системный промпт
	systemPrompt := LoadAIPrompt()

	// Добавляем данные о рынке, если доступны
	var marketDataText string
	if marketData != nil {
		marketDataText = s.marketDataService.FormatMarketDataForAI(marketData)
	} else {
		marketDataText = "ДАННЫЕ О РЫНКЕ НЕДОСТУПНЫ"
	}

	// Формируем сообщение с запросом на аналитику
	userPrompt := fmt.Sprintf("Сгенерируй актуальную аналитику по российскому фондовому рынку на сегодня. "+
		"Фокус на возможности инвестировать 1000 рублей. "+
		"Используй дружелюбный тон, добавь эмодзи. "+
		"Включи совет по инвестированию, который будет отличаться от предыдущих.\n\n"+
		"Вот текущие данные о рынке:\n\n%s", marketDataText)

	// Формируем запрос к API
	req := AIRequest{
		Model: s.modelName,
		Messages: []AIMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
	}

	// Преобразуем запрос в JSON
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	// Добавляем заголовки
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Парсим ответ
	var aiResp AIResponse
	err = json.Unmarshal(body, &aiResp)
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверка на ошибки в ответе API
	if aiResp.Error.Message != "" {
		return "", fmt.Errorf("ошибка API: %s", aiResp.Error.Message)
	}

	// Проверка наличия ответа
	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от API")
	}

	// Возвращаем содержимое ответа
	return aiResp.Choices[0].Message.Content, nil
}

// LoadAIPrompt загружает промпт для AI из файла
func LoadAIPrompt() string {
	// Попытка загрузки из файла
	content, err := os.ReadFile("ai_prompt.txt")
	if err == nil && len(content) > 0 {
		return string(content)
	}

	// Если файл не найден, используем стандартный промпт
	return `Ты финансовый аналитик, специализирующийся на российском фондовом рынке. 
Твоя задача - давать полезные и понятные советы по инвестированию начинающим инвесторам.
Говори простым и дружелюбным языком, избегай сложных терминов. 
Используй эмодзи, чтобы сделать текст более живым.
Включай конкретные рекомендации по акциям, которые стоит купить на 1000 рублей.
Подкрепляй свои рекомендации актуальными данными и трендами.
Добавляй уникальные советы по инвестированию, которые будут интересны и полезны новичкам.
Твои советы должны быть актуальными и учитывать реальную ситуацию на рынке.`
}
