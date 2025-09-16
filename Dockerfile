# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install minimal dependencies
RUN apk add --no-cache ca-certificates git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install ca-certificates and create user
RUN apk --no-cache add ca-certificates && \
    adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy binary and set ownership
COPY --from=builder /app/main .
RUN chown appuser:appuser main

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 1337

CMD ["./main"]