FROM golang:1.19 AS builder
WORKDIR /go/src/github.com/keyval-dev/odigos/odiglet
COPY . .
RUN apt-get update && apt-get install -y nasm
RUN nasm pkg/allocator/payload/mmap.asm
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app cmd/main.go

FROM gcr.io/distroless/static-debian10
WORKDIR /root/
COPY --from=builder /go/src/github.com/keyval-dev/odigos/odiglet/app .
CMD ["/root/app"]
