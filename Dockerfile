FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Adicione esta linha na seção de build (geralmente após o COPY . .)
RUN go install github.com/swaggo/swag/cmd/swag@latest

# E adicione este comando antes do RUN go build
RUN swag init -g cmd/api/main.go --output docs

RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service ./cmd/api

FROM alpine:latest
WORKDIR /app

COPY --from=builder /user-service .

RUN mkdir -p /app/configs
EXPOSE 8080
CMD ["./user-service"]