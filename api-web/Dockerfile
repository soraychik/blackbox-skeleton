FROM golang:1.23-alpine

WORKDIR /app

# Копируем сначала файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY *.go ./

# Собираем приложение
RUN go build -o /api-web

CMD [ "/api-web" ]