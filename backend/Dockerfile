FROM golang:1.24-alpine AS builder

WORKDIR /app

# Build the Go API binary from the api-go module.
COPY api-go/go.mod api-go/go.sum ./
RUN go mod download

COPY api-go/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o maas-api ./cmd/api

FROM alpine:3.19

WORKDIR /root/
COPY --from=builder /app/maas-api ./

EXPOSE 8000

CMD ["./maas-api"]
