FROM golang:alpine AS builder
LABEL authors="waraidako"

WORKDIR /bot

COPY . .

RUN CGO_ENABLED=0 go build -o bot .

ENTRYPOINT ["./bot"]