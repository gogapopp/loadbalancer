FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/loadbalancer ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/loadbalancer .
COPY configs/ ./configs/

ENTRYPOINT ["/app/loadbalancer"]