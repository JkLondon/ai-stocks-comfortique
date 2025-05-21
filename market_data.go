package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MarketData содержит данные о рынке для использования в аналитике
type MarketData struct {
	IndexMOEX        float64     `json:"index_moex"`
	IndexRTS         float64     `json:"index_rts"`
	USDRate          float64     `json:"usd_rate"`
	EURRate          float64     `json:"eur_rate"`
	TopStocks        []StockInfo `json:"top_stocks"`
	RecommendedStock StockInfo   `json:"recommended_stock"`
	MarketTrend      string      `json:"market_trend"` // "up", "down", "stable"
	MarketNews       []NewsItem  `json:"market_news"`
}

// StockInfo содержит информацию об акции
type StockInfo struct {
	Ticker    string  `json:"ticker"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Change    float64 `json:"change"` // изменение в процентах
	Currency  string  `json:"currency"`
	SourceURL string  `json:"source_url,omitempty"`
}

// NewsItem содержит новость о рынке
type NewsItem struct {
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

// MarketDataService предоставляет данные о рынке
type MarketDataService struct {
	client *http.Client
}

// NewMarketDataService создает новый экземпляр MarketDataService
func NewMarketDataService() *MarketDataService {
	return &MarketDataService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetMarketData получает актуальные данные о рынке
func (s *MarketDataService) GetMarketData() (*MarketData, error) {
	// Получаем данные Московской биржи
	moexData, err := s.getMOEXData()
	if err != nil {
		log.Printf("Ошибка при получении данных MOEX: %v", err)
		// Продолжаем, используя заглушку
		moexData = &MarketData{
			IndexMOEX:   4100.0, // Заглушка
			IndexRTS:    1100.0, // Заглушка
			USDRate:     90.5,   // Заглушка
			EURRate:     98.7,   // Заглушка
			MarketTrend: "stable",
		}
	}

	// Получаем новости о рынке
	news, err := s.getMarketNews()
	if err != nil {
		log.Printf("Ошибка при получении новостей: %v", err)
		// Продолжаем без новостей
		news = []NewsItem{}
	}
	moexData.MarketNews = news

	// Получаем топовые акции
	stocks, err := s.getTopStocks()
	if err != nil {
		log.Printf("Ошибка при получении данных о топовых акциях: %v", err)
		// Используем заглушку
		stocks = []StockInfo{
			{Ticker: "GAZP", Name: "Газпром", Price: 164.82, Change: 0.63, Currency: "RUB"},
			{Ticker: "SBER", Name: "Сбербанк", Price: 287.45, Change: 1.12, Currency: "RUB"},
			{Ticker: "LKOH", Name: "Лукойл", Price: 7046.5, Change: -0.35, Currency: "RUB"},
		}
	}
	moexData.TopStocks = stocks

	// Определяем рекомендуемую акцию (пример, в реальности нужен анализ)
	if len(stocks) > 0 {
		// Находим акцию с наибольшим ростом
		bestStock := stocks[0]
		for _, stock := range stocks {
			if stock.Change > bestStock.Change {
				bestStock = stock
			}
		}
		moexData.RecommendedStock = bestStock
	} else {
		// Заглушка
		moexData.RecommendedStock = StockInfo{
			Ticker:   "GAZP",
			Name:     "Газпром",
			Price:    164.82,
			Change:   0.63,
			Currency: "RUB",
		}
	}

	return moexData, nil
}

type MoexAPIResponse struct {
	Cbrf struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"cbrf"`
	WapRates struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"wap_rates"`
}

// getMOEXData получает данные с Мосбиржи
func (s *MarketDataService) getMOEXData() (*MarketData, error) {
	// URL API Московской Биржи для получения информации по индексам
	moexURL := "https://iss.moex.com/iss/engines/stock/markets/index/securities.json?iss.meta=off&iss.only=securities,marketdata"

	resp, err := s.client.Get(moexURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе к MOEX API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа MOEX API: %w", err)
	}

	// Парсим JSON
	var moexResp map[string]interface{}
	if err := json.Unmarshal(body, &moexResp); err != nil {
		return nil, fmt.Errorf("ошибка при парсинге ответа MOEX API: %w", err)
	}

	// Инициализируем MarketData
	marketData := &MarketData{
		MarketTrend: "stable", // По умолчанию считаем рынок стабильным
	}

	// Обрабатываем данные из ответа
	securities, ok := moexResp["securities"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат данных от MOEX API")
	}

	secData, ok := securities["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат data от MOEX API")
	}

	marketdata, ok := moexResp["marketdata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат marketdata от MOEX API")
	}

	mdData, ok := marketdata["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат marketdata.data от MOEX API")
	}

	// Ищем индексы MOEX и RTS
	for i, sec := range secData {
		secArray, ok := sec.([]interface{})
		if !ok || len(secArray) < 2 {
			continue
		}

		ticker, ok := secArray[0].(string)
		if !ok {
			continue
		}

		// Получаем соответствующие данные маркетдаты
		if i < len(mdData) {
			mdArray, ok := mdData[i].([]interface{})
			if !ok || len(mdArray) < 3 {
				continue
			}

			lastPrice, ok := mdArray[2].(float64)
			if !ok {
				// Пробуем преобразовать из строки
				lastPriceStr, ok := mdArray[2].(string)
				if !ok {
					continue
				}
				lastPrice, err = strconv.ParseFloat(lastPriceStr, 64)
				if err != nil {
					continue
				}
			}

			// Определяем индекс
			switch ticker {
			case "IMOEX":
				marketData.IndexMOEX = lastPrice
			case "RTSI":
				marketData.IndexRTS = lastPrice
			}
		}
	}

	// Получаем курсы валют
	currencyURL := "https://iss.moex.com/iss/statistics/engines/currency/markets/selt/rates.json?iss.meta=off"
	currResp, err := s.client.Get(currencyURL)
	if err != nil {
		log.Printf("Ошибка при запросе к MOEX API для валют: %v", err)
	} else {
		defer currResp.Body.Close()

		currBody, err := io.ReadAll(currResp.Body)
		if err == nil {
			var currResp MoexAPIResponse
			if err := json.Unmarshal(currBody, &currResp); err == nil {
				// Сопоставляем названия колонок с их индексами
				idx := make(map[string]int, len(currResp.Cbrf.Columns))
				for i, col := range currResp.Cbrf.Columns {
					idx[col] = i
				}

				// Берём первую строку данных
				row := currResp.Cbrf.Data[0]

				// Извлекаем и приводим типы
				marketData.USDRate, _ = row[idx["CBRF_USD_LAST"]].(float64)

				marketData.EURRate, _ = row[idx["CBRF_EUR_LAST"]].(float64)
			}
		}
	}

	// Определение тренда рынка на основе изменения индекса ММВБ
	// В реальном приложении нужен более сложный анализ
	marketData.MarketTrend = "stable"
	if marketData.IndexMOEX > 4200 {
		marketData.MarketTrend = "up"
	} else if marketData.IndexMOEX < 3800 {
		marketData.MarketTrend = "down"
	}

	return marketData, nil
}

// getTopStocks получает информацию о топовых акциях
func (s *MarketDataService) getTopStocks() ([]StockInfo, error) {
	// URL для получения данных о торгуемых акциях
	stocksURL := "https://iss.moex.com/iss/engines/stock/markets/shares/securities.json?iss.meta=off&iss.only=securities,marketdata&sort_column=VALTODAY&sort_order=desc&limit=20"

	resp, err := s.client.Get(stocksURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе к MOEX API для акций: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа MOEX API для акций: %w", err)
	}

	var moexResp map[string]interface{}
	if err := json.Unmarshal(body, &moexResp); err != nil {
		return nil, fmt.Errorf("ошибка при парсинге ответа MOEX API для акций: %w", err)
	}

	// Получаем индексы столбцов
	securities, ok := moexResp["securities"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат данных от MOEX API для акций")
	}

	columns, ok := securities["columns"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат columns от MOEX API для акций")
	}

	secData, ok := securities["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат data от MOEX API для акций")
	}

	marketdata, ok := moexResp["marketdata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат marketdata от MOEX API для акций")
	}

	mdColumns, ok := marketdata["columns"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат marketdata.columns от MOEX API для акций")
	}

	mdData, ok := marketdata["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("некорректный формат marketdata.data от MOEX API для акций")
	}

	// Определяем индексы нужных столбцов
	var tickerIdx, shortNameIdx, boardIdIdx int = -1, -1, -1
	for i, col := range columns {
		colName, ok := col.(string)
		if !ok {
			continue
		}
		switch colName {
		case "SECID":
			tickerIdx = i
		case "SHORTNAME":
			shortNameIdx = i
		case "BOARDID":
			boardIdIdx = i
		}
	}

	var lastIdx, changeIdx int = -1, -1
	for i, col := range mdColumns {
		colName, ok := col.(string)
		if !ok {
			continue
		}
		switch colName {
		case "LAST":
			lastIdx = i
		case "CHANGE":
			changeIdx = i
		}
	}

	// Проверяем, что нашли все нужные индексы
	if tickerIdx == -1 || shortNameIdx == -1 || lastIdx == -1 {
		return nil, fmt.Errorf("не удалось найти все необходимые столбцы в ответе API")
	}

	// Собираем акции
	stocks := []StockInfo{}
	stockMap := make(map[string]*StockInfo) // используем для объединения данных по одному тикеру

	// Сначала заполняем информацию из securities
	for i, sec := range secData {
		secArray, ok := sec.([]interface{})
		if !ok || len(secArray) <= shortNameIdx {
			continue
		}

		ticker, ok := secArray[tickerIdx].(string)
		if !ok {
			continue
		}

		// Проверяем, это акция с основной торговой площадки TQBR
		boardId, ok := secArray[boardIdIdx].(string)
		if !ok || boardId != "TQBR" {
			continue
		}

		name, ok := secArray[shortNameIdx].(string)
		if !ok {
			name = ticker
		}

		// Ищем соответствующие данные маркетдаты
		if i < len(mdData) {
			mdArray, ok := mdData[i].([]interface{})
			if !ok || len(mdArray) <= lastIdx {
				continue
			}

			// Получаем цену
			var price float64
			var change float64

			if lastIdx >= 0 && lastIdx < len(mdArray) {
				if priceVal, ok := mdArray[lastIdx].(float64); ok {
					price = priceVal
				} else if priceStr, ok := mdArray[lastIdx].(string); ok {
					if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
						price = p
					}
				}
			}

			if changeIdx >= 0 && changeIdx < len(mdArray) {
				if changeVal, ok := mdArray[changeIdx].(float64); ok {
					change = changeVal
				} else if changeStr, ok := mdArray[changeIdx].(string); ok {
					if c, err := strconv.ParseFloat(changeStr, 64); err == nil {
						change = c
					}
				}
			}

			stock := StockInfo{
				Ticker:   ticker,
				Name:     name,
				Price:    price,
				Change:   change,
				Currency: "RUB",
			}

			stockMap[ticker] = &stock
		}
	}

	// Преобразуем map в slice
	for _, stock := range stockMap {
		stocks = append(stocks, *stock)
	}

	// Если мы не получили никаких акций, возвращаем ошибку
	if len(stocks) == 0 {
		return nil, fmt.Errorf("не удалось получить информацию об акциях")
	}

	// Ограничиваем список до 5 акций
	if len(stocks) > 5 {
		stocks = stocks[:5]
	}

	return stocks, nil
}

// getMarketNews получает последние новости о рынке
func (s *MarketDataService) getMarketNews() ([]NewsItem, error) {
	// В реальном приложении здесь будет запрос к API новостей
	// Для примера используем NewsAPI или заглушку

	// Заглушка для новостей
	news := []NewsItem{
		{
			Title:     "Индекс Мосбиржи: актуальный прогноз на сегодня",
			Source:    "РБК",
			URL:       "https://www.rbc.ru/finances/",
			Timestamp: time.Now().Add(-2 * time.Hour),
		},
		{
			Title:     "Какие акции российских компаний показывают рост в текущих условиях",
			Source:    "Ведомости",
			URL:       "https://www.vedomosti.ru/finance/",
			Timestamp: time.Now().Add(-5 * time.Hour),
		},
		{
			Title:     "Курс рубля: факторы влияния и перспективы на ближайшее время",
			Source:    "Коммерсантъ",
			URL:       "https://www.kommersant.ru/finance/",
			Timestamp: time.Now().Add(-24 * time.Hour), // 1 день
		},
	}

	// Пытаемся получить реальные новости
	newsAPI := os.Getenv("NEWS_API_KEY")
	if newsAPI != "" {
		realNews, err := s.fetchNewsFromAPI(newsAPI)
		if err == nil && len(realNews) > 0 {
			return realNews, nil
		}
	}

	return news, nil
}

// fetchNewsFromAPI получает новости из API новостей
func (s *MarketDataService) fetchNewsFromAPI(apiKey string) ([]NewsItem, error) {
	// URL для запроса к NewsAPI
	newsURL := fmt.Sprintf("https://newsapi.org/v2/everything?q=российский+фондовый+рынок+акции&language=ru&apiKey=%s&pageSize=3", apiKey)

	resp, err := s.client.Get(newsURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе к News API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа News API: %w", err)
	}

	var newsResp struct {
		Status   string `json:"status"`
		Articles []struct {
			Title  string `json:"title"`
			Source struct {
				Name string `json:"name"`
			} `json:"source"`
			URL         string `json:"url"`
			PublishedAt string `json:"publishedAt"`
		} `json:"articles"`
	}

	if err := json.Unmarshal(body, &newsResp); err != nil {
		return nil, fmt.Errorf("ошибка при парсинге ответа News API: %w", err)
	}

	if newsResp.Status != "ok" || len(newsResp.Articles) == 0 {
		return nil, fmt.Errorf("некорректный ответ от News API")
	}

	var news []NewsItem
	for _, article := range newsResp.Articles {
		// Парсим время публикации
		pubTime, err := time.Parse(time.RFC3339, article.PublishedAt)
		if err != nil {
			pubTime = time.Now()
		}

		news = append(news, NewsItem{
			Title:     article.Title,
			Source:    article.Source.Name,
			URL:       article.URL,
			Timestamp: pubTime,
		})
	}

	return news, nil
}

// FormatMarketDataForAI форматирует данные о рынке для использования в запросе к AI
func (s *MarketDataService) FormatMarketDataForAI(data *MarketData) string {
	var sb strings.Builder

	// Индексы и курсы валют
	sb.WriteString(fmt.Sprintf("📊 ИНДЕКСЫ:\n"))
	sb.WriteString(fmt.Sprintf("- Индекс Мосбиржи: %.2f\n", data.IndexMOEX))
	sb.WriteString(fmt.Sprintf("- Индекс РТС: %.2f\n", data.IndexRTS))
	sb.WriteString(fmt.Sprintf("- Курс USD/RUB: %.2f\n", data.USDRate))
	sb.WriteString(fmt.Sprintf("- Курс EUR/RUB: %.2f\n\n", data.EURRate))

	// Тренд рынка
	sb.WriteString(fmt.Sprintf("🔍 ТРЕНД РЫНКА: %s\n\n", translateTrend(data.MarketTrend)))

	// Топ акции
	sb.WriteString("🏆 ТОП АКЦИИ:\n")
	for _, stock := range data.TopStocks {
		change := ""
		if stock.Change > 0 {
			change = fmt.Sprintf("+%.2f%%", stock.Change)
		} else {
			change = fmt.Sprintf("%.2f%%", stock.Change)
		}
		sb.WriteString(fmt.Sprintf("- %s (%s): %.2f %s (%s)\n",
			stock.Name, stock.Ticker, stock.Price, stock.Currency, change))
	}
	sb.WriteString("\n")

	// Рекомендуемая акция
	sb.WriteString("💎 РЕКОМЕНДАЦИЯ:\n")
	sb.WriteString(fmt.Sprintf("- %s (%s): %.2f %s (изменение: %.2f%%)\n\n",
		data.RecommendedStock.Name, data.RecommendedStock.Ticker,
		data.RecommendedStock.Price, data.RecommendedStock.Currency,
		data.RecommendedStock.Change))

	// Новости рынка
	sb.WriteString("📰 ПОСЛЕДНИЕ НОВОСТИ:\n")
	for _, news := range data.MarketNews {
		sb.WriteString(fmt.Sprintf("- %s (Источник: %s, %s)\n",
			news.Title, news.Source, news.Timestamp.Format("02.01.2006")))
	}
	println(sb.String())
	return sb.String()
}

// translateTrend переводит тренд на русский
func translateTrend(trend string) string {
	switch trend {
	case "up":
		return "РОСТ 📈"
	case "down":
		return "ПАДЕНИЕ 📉"
	default:
		return "СТАБИЛЬНЫЙ ↔️"
	}
}

// Вспомогательная функция для форматирования JSON
func prettyPrintJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonBytes)
}
