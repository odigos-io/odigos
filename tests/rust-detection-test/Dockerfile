FROM rust:1.75-bookworm AS builder

WORKDIR /app
COPY Cargo.toml .
COPY src src

RUN cargo build --release

RUN cp target/release/rust-detection-test /app/rust-app-unstripped

RUN strip --strip-all target/release/rust-detection-test -o /app/rust-app-stripped

FROM debian:bookworm-slim AS unstripped
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/rust-app-unstripped /app/server
EXPOSE 8080
CMD ["/app/server"]

FROM debian:bookworm-slim AS stripped
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/rust-app-stripped /app/server
EXPOSE 8080
CMD ["/app/server"]

