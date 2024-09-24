# Stage 1: Dependencies
FROM --platform=$BUILDPLATFORM node:20.2.0 AS deps
WORKDIR /app
# Copy necessary Yarn files for PnP
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./
COPY frontend/webapp/.yarn ./.yarn
COPY frontend/webapp/.yarnrc.yml ./.yarnrc.yml
# Use cache to speed up installation
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn/v6 \
    yarn install --frozen-lockfile


FROM --platform=$BUILDPLATFORM node:20.2.0 AS builder
WORKDIR /webapp
COPY frontend//webapp .
RUN yarn build --verbose


FROM --platform=$BUILDPLATFORM golang:1.22 AS backend
WORKDIR /app
COPY . .
COPY --from=builder /webapp/out frontend/webapp/out
WORKDIR /app/frontend
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o odigos-ui

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=backend /app/frontend/odigos-ui .
USER 65532:65532
ENTRYPOINT ["/app/odigos-ui"]