FROM golang:alpine AS builder
LABEL authors="waraidako"

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 go build -o /bot .

FROM alpine
RUN apk add --no-cache ca-certificates

COPY --from=builder /bot /bot

ENTRYPOINT ["/bot"]