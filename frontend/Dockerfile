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

# Stage 2: Build
FROM --platform=$BUILDPLATFORM node:20.2.0 AS builder
WORKDIR /webapp
# Copy all necessary files, including Yarn PnP setup
COPY frontend/webapp .
COPY --from=deps /app/.yarn ./.yarn
COPY --from=deps /app/.pnp.cjs ./.pnp.cjs
COPY --from=deps /app/.yarnrc.yml ./.yarnrc.yml
# Install dependencies to ensure PnP setup works in this stage
RUN yarn install --frozen-lockfile
# Set NODE_OPTIONS to use PnP loader
ENV NODE_OPTIONS="--require ./.pnp.cjs"
# Attempt to build the project
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