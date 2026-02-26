# syntax=docker/dockerfile:1

FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nurmed ./cmd/nurmed

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && install -d -m 755 -o app -g app /app

WORKDIR /app

COPY --from=builder /src/nurmed /app/nurmed
COPY configs /app/configs
COPY migrations /app/migrations
copy api /app/api

ENV GIN_MODE=release

EXPOSE 9050

USER app

CMD ["./nurmed"]
