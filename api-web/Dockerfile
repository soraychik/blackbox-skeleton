FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем сначала файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o api-web .

# Используем scratch image для финального образа
FROM scratch

WORKDIR /

# Копируем бинарник из builder stage
COPY --from=0 /app/api-web .

CMD [ "/api-web" ]