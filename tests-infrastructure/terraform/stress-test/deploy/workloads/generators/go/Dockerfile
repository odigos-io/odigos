# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-span-gen .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/go-span-gen .

# Set environment variables
ENV OTEL_SERVICE_NAME=go-span-gen
ENV OTEL_RESOURCE_ATTRIBUTES=service.name=go-span-gen
ENV SPANS_PER_SEC=5000
ENV SPAN_BYTES=10000

# Run the application
CMD ["./go-span-gen"]
