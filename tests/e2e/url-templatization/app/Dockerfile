FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o url-demo-app main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/url-demo-app .
EXPOSE 8080
CMD ["./url-demo-app"]
