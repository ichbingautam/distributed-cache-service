# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for fetch (if needed)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

# Final stage
FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/server .

# Create directory for Raft data
RUN mkdir -p /app/raft_data

# Expose HTTP port
EXPOSE 8000

# Run the server
# Run the server
CMD ["./server", "-bootstrap"]

# OCI Labels
LABEL org.opencontainers.image.source=https://github.com/ichbingautam/distributed-cache-service
LABEL org.opencontainers.image.description="Distributed Cache Service"
LABEL org.opencontainers.image.licenses=MIT
