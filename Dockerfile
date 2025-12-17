# Этап сборки
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем исходный код
COPY app/ .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o imagehost .

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Создаем директорию для данных
RUN mkdir -p /data

# Копируем бинарник из этапа сборки
COPY --from=builder /app/imagehost .

# Копируем шаблоны
COPY --from=builder /app/templates ./templates

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./imagehost"]