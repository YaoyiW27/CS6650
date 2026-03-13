FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o order-receiver .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-receiver .
EXPOSE 8080
CMD ["./order-receiver"]