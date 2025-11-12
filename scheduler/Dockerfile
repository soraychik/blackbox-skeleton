FROM golang:1.21-alpine

WORKDIR /app

# Копируем файлы зависимостей сначала
COPY go.mod go.sum ./
RUN go mod download

# Копируем всю папку scheduler
COPY . .

# Собираем приложение
RUN go build -o /scheduler

CMD [ "/scheduler" ]