FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o transcoder-service ./cmd/server

# Use a multi-stage build to create a smaller image
FROM alpine:3.18

# Install FFmpeg
RUN apk add --no-cache ffmpeg

# Create a non-root user
RUN adduser -D -g '' appuser

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/transcoder-service .

# Create the temp directory
RUN mkdir -p /tmp/transcoder && chown -R appuser:appuser /tmp/transcoder

# Switch to the non-root user
USER appuser

# Expose the port
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/app/transcoder-service"] 