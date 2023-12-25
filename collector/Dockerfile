FROM golang:1.21-bullseye AS builder
COPY . /go/src/
WORKDIR /go/src/
RUN make genodigoscol

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/odigosotelcol/odigosotelcol /odigosotelcol
CMD ["/odigosotelcol"]