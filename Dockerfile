# Build stage
FROM golang:1.21-alpine AS builder

# Install ca-certificates for go mod download
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /irrigation-analytics ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates openssl

# Copy binary from builder
COPY --from=builder /irrigation-analytics .

# Generate self-signed certificate for development
RUN openssl req -x509 -newkey rsa:4096 -keyout /app/key.pem -out /app/cert.pem -days 365 -nodes -subj "/CN=localhost"

# Expose port
EXPOSE 8443

# Set environment variables
ENV TLS_CERT=/app/cert.pem
ENV TLS_KEY=/app/key.pem
ENV PORT=8443

# Run the application
CMD ["./irrigation-analytics"]
