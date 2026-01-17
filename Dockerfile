# Build stage
FROM golang:alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o tg-downloader .

# Final stage
FROM alpine:latest

# Install runtime dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/tg-downloader .

# The application expects a config.yaml in the same directory
# We don't copy it here to allow users to mount it as a volume
# COPY config.yaml . 

# Command to run
ENTRYPOINT ["./tg-downloader"]
