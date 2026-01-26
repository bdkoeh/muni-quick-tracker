FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code
COPY main.go .
COPY go.mod .

# Download dependencies and build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o muni-tracker main.go

# Runtime image
FROM alpine:latest

WORKDIR /app

# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary and static files
COPY --from=builder /app/muni-tracker .
COPY static/ ./static/

# Expose port
EXPOSE 8080

# Run
CMD ["./muni-tracker"]
