FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ---- Runtime ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /app/bin/server .

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

CMD ["./server"]
