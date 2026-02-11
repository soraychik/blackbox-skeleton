FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей сначала
COPY go.mod go.sum ./
RUN go mod download

# Копируем всю папку scheduler
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler .

# Используем Alpine для финального образа (нужен для монтирования NFS/SMB)
FROM alpine:latest

WORKDIR /

# Устанавливаем пакеты для работы с сетевыми файловыми системами
RUN apk add --no-cache \
    nfs-utils \
    cifs-utils \
    coreutils \
    ca-certificates

# Копируем бинарник из builder stage
COPY --from=0 /app/scheduler .

# Создаем директорию для конфигов
RUN mkdir -p /app/configs

CMD [ "/scheduler" ]