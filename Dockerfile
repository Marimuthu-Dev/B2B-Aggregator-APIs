# Build stage (Go 1.25 to match go.mod)
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files first for better layer caching
COPY src/go.mod src/go.sum ./
RUN go mod download

# Copy source and build optimized binary
COPY src/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /api ./cmd/api

# Final stage: minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /api .

EXPOSE 8080

CMD ["./api"]
