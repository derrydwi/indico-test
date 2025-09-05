# Build stage
FROM golang:1.25-alpine AS builder

# Install git and ca-certificates (needed for downloading Go modules)
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create appuser for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main cmd/server/main.go

# Build seeder binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o seeder cmd/seeder/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create appuser for security
RUN adduser -D -g '' appuser

# Create /tmp/settlements directory with proper permissions
RUN mkdir -p /tmp/settlements && chown appuser:appuser /tmp/settlements

# Copy binaries
COPY --from=builder /app/main /go/bin/main
COPY --from=builder /app/seeder /go/bin/seeder

# Switch to appuser
USER appuser

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/go/bin/main"]
