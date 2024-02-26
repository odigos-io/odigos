FROM golang:1.21-bullseye AS builder
COPY . /go/src/
WORKDIR /go/src/odigosotelcol
RUN go build -o odigosotelcol -trimpath -ldflags="-s -w" .

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/odigosotelcol/odigosotelcol /odigosotelcol
CMD ["/odigosotelcol"]