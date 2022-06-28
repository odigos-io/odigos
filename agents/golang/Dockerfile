FROM fedora:35 as builder
RUN dnf install clang llvm make libbpf-devel git -y
RUN curl -LO https://go.dev/dl/go1.18.linux-amd64.tar.gz && tar -C /usr/local -xzf go*.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
RUN git clone https://github.com/keyval-dev/opentelemetry-go-instrumentation /app
WORKDIR /app
RUN make build

FROM gcr.io/distroless/base-debian11
COPY --from=builder /app/kv-go-instrumentation /
CMD ["/kv-go-instrumentation"]