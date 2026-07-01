FROM --platform=$BUILDPLATFORM node:20.19.0 AS deps
WORKDIR /app
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./
COPY frontend/webapp/.yalc* ./.yalc/

# This allows us to deploy the UI to local clusters with the yalc-linked version of the UI kit.
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn \
    if [ -d .yalc ] && [ "$(ls -A .yalc 2>/dev/null)" ]; then \
      echo "Using yalc"; \
      npm install -g yalc; \
      yarn install; \
    else \
      echo "Using frozen lockfile"; \
      yarn install --frozen-lockfile; \
    fi

FROM --platform=$BUILDPLATFORM node:20.19.0 AS builder
WORKDIR /webapp
COPY --from=deps /app/node_modules ./node_modules
COPY frontend/webapp .
RUN yarn build

FROM --platform=$BUILDPLATFORM golang:1.26.2 AS backend
WORKDIR /app
# Copy only the go.mod/go.sum of the frontend module and its locally-replaced
# deps first. This lets `go mod download` be cached independently of source
# changes, so editing source doesn't re-download the (large) dependency tree.
COPY frontend/go.mod frontend/go.sum ./frontend/
COPY api/go.mod api/go.sum ./api/
COPY common/go.mod common/go.sum ./common/
COPY config/go.mod config/go.sum ./config/
COPY destinations/go.mod destinations/go.sum ./destinations/
COPY actions/go.mod actions/go.sum ./actions/
COPY k8sutils/go.mod k8sutils/go.sum ./k8sutils/
COPY odigosauth/go.mod odigosauth/go.sum ./odigosauth/
RUN --mount=type=cache,target=/go/pkg/mod cd frontend && go mod download

# Copy the source for the frontend module and the modules it replaces locally
# (see the `replace` directives in frontend/go.mod and its transitive deps).
# Scoping the copy to just these modules means unrelated changes elsewhere in
# the monorepo no longer bust the build cache.
COPY api ./api
COPY common ./common
COPY config ./config
COPY destinations ./destinations
COPY actions ./actions
COPY k8sutils ./k8sutils
COPY odigosauth ./odigosauth
COPY frontend ./frontend
# Strip the webapp source tree but keep embed.go — the frontend/webapp Go
# package embeds the built bundle (//go:embed all:out) and the server package
# imports it, so deleting it breaks the build.
RUN find frontend/webapp -mindepth 1 -maxdepth 1 ! -name embed.go -exec rm -rf {} +
COPY --from=builder /webapp/out frontend/webapp/out
WORKDIR /app/frontend
ARG TARGETARCH
ARG LD_FLAGS
ARG RHEL=false
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build,id=go-build-$TARGETARCH \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="${LD_FLAGS}" -o odigos-ui
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build,id=go-build-$TARGETARCH \
    if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="frontend"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
WORKDIR /app
COPY --from=backend /app/frontend/odigos-ui .
COPY --from=backend /app/frontend/licenses /licenses
COPY --from=backend /app/frontend/LICENSE /licenses/.
USER 65532:65532
ENTRYPOINT ["/app/odigos-ui", "--address", "0.0.0.0"]

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=backend /app/frontend/odigos-ui .
USER 65532:65532
ENTRYPOINT ["/app/odigos-ui", "--address", "0.0.0.0"]
