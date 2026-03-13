# ── development ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS dev

WORKDIR /app

# Air for hot reload
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

# Source is mounted as a volume — no COPY needed here
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]


# ── build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api


# ── production ────────────────────────────────────────────────────────────────
FROM alpine:3.19 AS production

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/api .

EXPOSE 8080
ENTRYPOINT ["./api"]