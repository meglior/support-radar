# Stage 1: Сборка приложения
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статически скомпилированный бинарник под Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o support-radar-server ./cmd/server/main.go

# Stage 2: Финальный минимальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копируем бинарник из предыдущего шага
COPY --from=builder /app/support-radar-server .
# Копируем конфигурационный файл (если он нужен рядом)
COPY --from=builder /app/config.yaml .

# Открываем порт для mTLS
EXPOSE 8443

CMD ["./support-radar-server"]