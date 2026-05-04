FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    $(go env GOPATH)/bin/swag init --parseDependency --parseInternal
RUN go build -o main .

FROM alpine

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs

EXPOSE 3000

CMD ["./main"]