FROM debian:bookworm-slim AS rsync-builder
ARG RSYNC_VERSION=3.5.1
ARG RSYNC_SHA256=c55f9c9dc10fb8bec397b399a0fdded53cc9a2d8e30891bb0d63724d25c37bef
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
    && rm -rf /var/lib/apt/lists/*

# rsync is only used for local copies, so it is built without OpenSSL (it falls back to
# its built-in MD4/MD5 and xxhash checksums) and without IDN hostname support.
RUN wget https://download.samba.org/pub/rsync/src/rsync-${RSYNC_VERSION}.tar.gz \
    && echo "${RSYNC_SHA256}  rsync-${RSYNC_VERSION}.tar.gz" | sha256sum -c - \
    && tar -xzf rsync-${RSYNC_VERSION}.tar.gz \
    && cd rsync-${RSYNC_VERSION} \
    && ./configure --prefix=/usr --disable-openssl --disable-idn LDFLAGS="-static" \
    && make -j"$(nproc)" \
    && make install DESTDIR=/rsync-install \
    && cd .. \
    && rm -rf rsync-${RSYNC_VERSION}*

FROM golang:1.26.6-trixie

# goreleaser is used to build vmagent
RUN echo "deb [trusted=yes] https://repo.goreleaser.com/apt/ /" > /etc/apt/sources.list.d/goreleaser.list
RUN apt-get update && apt-get install -y \
    curl \
    clang \
    gcc \
    llvm \
    make \
    libbpf-dev \
    goreleaser \
    libcap2-bin

# Bring in rsync
COPY --from=rsync-builder /rsync-install/usr/bin/rsync /usr/bin/rsync
