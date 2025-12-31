# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for fetch (if needed)
# Install build tools
RUN apk add --no-cache git build-base curl

COPY go.mod go.sum ./
RUN go mod download

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.0

COPY . .

# Run CI Steps (Replicating GitHub Workflow)
# 1. Vet
RUN go vet ./...
# 2. Lint
RUN golangci-lint run ./...
# 3. Test
RUN CGO_ENABLED=0 go test -v ./...

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
