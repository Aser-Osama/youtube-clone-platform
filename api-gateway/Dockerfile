FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./cmd/main.go

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/keys ./keys

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./api-gateway"] 