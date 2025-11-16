# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/main .
# Copy templates
COPY --from=builder /app/templates ./templates
# Create necessary directories
RUN mkdir -p static/photos static/photos/thumbs

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"] 