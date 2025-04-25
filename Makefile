.PHONY: build run docker-build docker-run docker-up docker-down docker-logs

# Имя приложения
APP_NAME = ai-stocks-bot

# Основные команды сборки и запуска
build:
	go build -o $(APP_NAME) .

run: build
	./$(APP_NAME)

# Docker команды
docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run -d --name $(APP_NAME) $(APP_NAME):latest

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Очистка
clean:
	rm -f $(APP_NAME)
	go clean

# Инициализация проекта
setup:
	go mod tidy
	cp config_example.env .env
	@echo "Настройте файл .env с вашими данными для работы бота"

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  make build         - Сборка бинарного файла"
	@echo "  make run           - Сборка и запуск приложения"
	@echo "  make docker-build  - Сборка Docker образа"
	@echo "  make docker-run    - Запуск Docker контейнера"
	@echo "  make docker-up     - Запуск с использованием docker-compose"
	@echo "  make docker-down   - Остановка контейнеров docker-compose"
	@echo "  make docker-logs   - Просмотр логов docker-compose"
	@echo "  make clean         - Очистка бинарных файлов"
	@echo "  make setup         - Инициализация проекта" 