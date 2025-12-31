# Build stage
FROM golang:1.21-alpine AS builder

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

# Expose HTTP and Raft ports
EXPOSE 8080 11000

# Run the server
CMD ["./server"]
