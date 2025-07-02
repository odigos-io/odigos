FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o loadgen main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

COPY --from=builder /app/loadgen .

# Add health check script
RUN echo '#!/bin/sh' > /health-check.sh && \
    echo 'curl -f http://localhost:8080/health || exit 1' >> /health-check.sh && \
    chmod +x /health-check.sh

EXPOSE 8080

CMD ["./loadgen"]