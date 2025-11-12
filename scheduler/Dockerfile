FROM golang:1.21-alpine
WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go build -o /scheduler
CMD [ "/scheduler" ]