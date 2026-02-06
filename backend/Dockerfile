# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
# (gcc/musl-dev needed for CGO if we used SQLite, but BadgerDB is pure Go usually. 
# However, race detection or other net tools might need it. Safe to have.)
RUN apk add --no-cache git build-base

# Copy Module files
COPY go.mod go.sum ./
RUN go mod download

# Copy Source
COPY . .

# Build Binary
# CGO_ENABLED=0 for static binary (Alpine compatible)
RUN CGO_ENABLED=0 GOOS=linux go build -o /node .

# Final Stage (Minimal)
FROM alpine:latest

WORKDIR /root/

# Install CA Certs for HTTPS requests
RUN apk --no-cache add ca-certificates

COPY --from=builder /node .

# Expose P2P and API ports
EXPOSE 3000 8080

# Create data directory for volume mount
RUN mkdir -p ./data

# Default Command (Can be overridden by Fly.toml)
CMD ["./node", "--port", "3000", "--mode", "full", "--vault", "./data/vault"]
