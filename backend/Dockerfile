# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies if needed (e.g., git, gcc)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN apk add --no-cache tzdata && go mod download

COPY . .

RUN go build -ldflags="-s -w" -o monitor-backend ./cmd/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Add CA certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/monitor-backend .

EXPOSE 8082

CMD ["./monitor-backend"]
