FROM --platform=$BUILDPLATFORM node:20.9.0 AS deps
WORKDIR /app
COPY frontend/webapp/package.json frontend/webapp/package-lock.json ./
RUN npm ci

FROM --platform=$BUILDPLATFORM node:20.9.0 AS builder
WORKDIR /webapp
COPY --from=deps /app/node_modules ./node_modules
COPY frontend/webapp .
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.22 AS backend
WORKDIR /app
COPY . .
COPY --from=builder /webapp/out frontend/webapp/out
COPY --from=builder /webapp/dep-out frontend/webapp/dep-out
WORKDIR /app/frontend
ENV GOMAXPROCS=4
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOARCH=$TARGETARCH \
    go build -trimpath -mod=readonly -o odigos-ui

FROM scratch
WORKDIR /app
COPY --from=backend /app/frontend/odigos-ui .
ENTRYPOINT ["/app/odigos-ui", "--address", "0.0.0.0"]
