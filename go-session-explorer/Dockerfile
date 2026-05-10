# Build stage
FROM golang:alpine AS builder

# Install git for dependency resolution if necessary
RUN apk add --no-cache git

WORKDIR /src

# Note: This Dockerfile must be built from the project root to satisfy
# the local 'replace' directive in go-session-explorer/go.mod:
#   replace github.com/sipeed/picoclaw => ../
# Command: docker build -t go-session-explorer -f go-session-explorer/Dockerfile .
COPY . .

WORKDIR /src/go-session-explorer

# Download dependencies
RUN go mod download

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o session-explorer .

# Final stage
FROM alpine:latest

# Install CA certificates for secure connections
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary and UI assets from the builder stage
COPY --from=builder /src/go-session-explorer/session-explorer .
COPY --from=builder /src/go-session-explorer/templates ./templates
COPY --from=builder /src/go-session-explorer/static ./static

# Expose the default port
EXPOSE 3001

# Configuration environment variables
ENV PICOCLAW_HOME=/data
ENV PORT=3001

# Setup persistent volume for session storage
RUN mkdir -p /data
VOLUME /data

# Start the application
ENTRYPOINT ["./session-explorer"]
