FROM golang:1.20-alpine AS builder

# Установка необходимых зависимостей
RUN apk add --no-cache git

# Создание рабочей директории
WORKDIR /app

# Копирование файлов для управления зависимостями
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-stocks-bot .

# Второй этап с использованием минимального образа
FROM alpine:latest

# Установка timezone и сертификатов
RUN apk --no-cache add ca-certificates tzdata

# Создание непривилегированного пользователя
RUN adduser -D -g '' appuser

# Установка рабочей директории
WORKDIR /app

# Копирование скомпилированного бинарного файла из этапа сборки
COPY --from=builder /app/ai-stocks-bot .
COPY --from=builder /app/ai_prompt.txt ./ai_prompt.txt
COPY --from=builder /app/config_example.env .env

# Указание, что файлами владеет непривилегированный пользователь
RUN chown -R appuser:appuser /app

# Переключение на непривилегированного пользователя
USER appuser

# Запуск приложения
CMD ["./ai-stocks-bot"] 