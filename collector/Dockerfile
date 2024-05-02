FROM golang:1.22-bullseye AS builder
COPY . /go/src/
WORKDIR /go/src/
RUN make build-odigoscol

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/odigosotelcol/odigosotelcol /odigosotelcol
CMD ["/odigosotelcol"]