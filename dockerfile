FROM golang:1.23.4-alpine AS builder

WORKDIR /usr/local/src

# Копируем зависимости
COPY app/go.mod app/go.sum ./
RUN go mod download

# Копируем исходники и и билдим
COPY app ./
RUN go build -ldflags="-s -w" -o /app cmd/main.go

# Стадия выполнения
FROM alpine AS runner

WORKDIR /root/

# Добавляем бинарный файл и миграции с конфигом
COPY --from=builder /app ./app
COPY config.yaml ./
COPY app/gates/storage/migrations ./migrations

CMD ["./app"]