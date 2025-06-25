FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG VERSION
ARG SHORT_COMMIT
ARG DATE

WORKDIR /src

# -------- 1) Copy ONLY go.mod/sum files for cache-friendly deps --------
COPY api/go.mod            api/go.sum            ./api/
COPY autoscaler/go.mod     autoscaler/go.sum     ./autoscaler/
COPY common/go.mod         common/go.sum         ./common/
COPY k8sutils/go.mod       k8sutils/go.sum       ./k8sutils/
COPY profiles/go.mod       profiles/go.sum       ./profiles/
COPY cli/go.mod            cli/go.sum            ./cli/

WORKDIR /src/cli
RUN go mod download

# -------- 2) Now copy the full source trees ----------------------------
WORKDIR /src/
COPY api/        ./api/
COPY autoscaler/ ./autoscaler/
COPY common/     ./common/
COPY k8sutils/   ./k8sutils/
COPY profiles/   ./profiles/
COPY cli/        ./cli/

WORKDIR /src/cli
RUN CGO_ENABLED=0 GOOS=linux \
    go build -tags=embed_manifests -o odigos .

# -------- 3) Minimal final image --------------------------------------
FROM scratch
COPY --from=builder /src/cli/odigos /usr/local/bin/odigos
ENTRYPOINT ["/usr/local/bin/odigos"]
