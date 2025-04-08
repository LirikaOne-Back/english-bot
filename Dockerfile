FROM golang:1.24-alpine AS builder

WORKDIR /app

# Установка необходимых зависимостей
RUN apk add --no-cache git

# Копирование файлов проекта
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o english_bot ./cmd/bot

# Финальный образ
FROM alpine:3.18

WORKDIR /app

# Установка часового пояса
RUN apk add --no-cache tzdata
ENV TZ=Europe/Moscow

# Копирование бинарника из предыдущего этапа
COPY --from=builder /app/english_bot .

# Создание директории для логов
RUN mkdir -p /app/logs

# Запуск приложения
CMD ["./english_bot"]