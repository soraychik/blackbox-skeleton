FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей сначала
COPY go.mod go.sum ./
RUN go mod download

# Копируем всю папку scheduler
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler .

# Используем scratch image для финального образа
FROM scratch

WORKDIR /

# Копируем бинарник из builder stage
COPY --from=0 /app/scheduler .

CMD [ "/scheduler" ]