# =========================
# BUILD STAGE
# =========================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies separately so Docker can cache them.
COPY go.mod go.sum ./
RUN go mod download

# Copy application source.
COPY . .

# Build a small production Linux binary.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/music-api \
    ./cmd/server


# =========================
# PRODUCTION STAGE
# =========================
FROM alpine:3.22

WORKDIR /app

# TLS certificates are required for HTTPS connections
# to services such as Neon and Cloudinary.
RUN apk --no-cache add ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

# Copy only what the running application needs.
COPY --from=builder /app/bin/music-api /app/music-api
COPY --from=builder /app/web /app/web

USER app

EXPOSE 7000

CMD ["/app/music-api"]
