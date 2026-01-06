FROM rust:1.83-bookworm AS builder

WORKDIR /app
COPY Cargo.toml .
COPY src src

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-otel-app /app/server
EXPOSE 8080
CMD ["/app/server"]

