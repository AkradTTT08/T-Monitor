FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o monitor-backend ./cmd/main.go

EXPOSE 5273

CMD ["./monitor-backend"]
