# --- Build stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

# ca-certificates for outbound HTTPS calls; tzdata so Asia/Kolkata business-date
# handling (internal/shared/dates.go) resolves the real IANA zone instead of
# falling back to its fixed-offset approximation.
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

# Versioned SQL migrations are applied at startup via a relative "file://migrations"
# path (internal/database/migrate.go) — must live alongside the binary's WORKDIR.
COPY --from=builder /out/server ./server
COPY --from=builder /app/migrations ./migrations

# Uploaded donor/bank documents (internal/upload) — mount a volume here so
# files survive container restarts/redeploys.
RUN mkdir -p uploads && chown -R app:app /app
VOLUME ["/app/uploads"]

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["./server"]
