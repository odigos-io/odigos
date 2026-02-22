FROM debian:bookworm-slim AS rsync-builder
ARG RSYNC_VERSION=3.2.7
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    wget \
    ca-certificates \
    libacl1-dev \
    libattr1-dev \
    libpopt-dev \
    liblz4-dev \
    libzstd-dev \
    libxxhash-dev \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://download.samba.org/pub/rsync/src/rsync-${RSYNC_VERSION}.tar.gz \
    && tar -xzf rsync-${RSYNC_VERSION}.tar.gz \
    && cd rsync-${RSYNC_VERSION} \
    && ./configure --prefix=/usr LDFLAGS="-static" \
    && make \
    && make install DESTDIR=/rsync-install \
    && cd .. \
    && rm -rf rsync-${RSYNC_VERSION}*

FROM golang:1.26.0-trixie

# goreleaser is used to build vmagent
RUN echo "deb [trusted=yes] https://repo.goreleaser.com/apt/ /" > /etc/apt/sources.list.d/goreleaser.list
RUN apt-get update && apt-get install -y \
    curl \
    clang \
    gcc \
    llvm \
    make \
    libbpf-dev \
    goreleaser

# Bring in rsync
COPY --from=rsync-builder /rsync-install/usr/bin/rsync /usr/bin/rsync
