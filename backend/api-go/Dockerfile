FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o smail-api ./cmd/api

FROM alpine:3.19

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/smail-api .

# Expose port
EXPOSE 8000

# Run the API
CMD ["./smail-api"]
