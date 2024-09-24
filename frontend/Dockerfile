# Stage 1: Dependencies
FROM --platform=$BUILDPLATFORM node:20.2.0 AS deps
WORKDIR /app
# Copy only the necessary files for installing dependencies
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./
# Use cache to speed up installation
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn/v6 \
    yarn install --frozen-lockfile --prefer-offline

# Stage 2: Build
FROM --platform=$BUILDPLATFORM node:20.2.0 AS builder
WORKDIR /webapp
# Copy the entire project
COPY frontend/webapp .
# No need to copy node_modules, as we are using Yarn PnP
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn/v6 \
    yarn build

# Stage 3: Backend (Golang)
FROM --platform=$BUILDPLATFORM golang:1.22 AS backend
WORKDIR /app
# Copy all backend files
COPY . .
# Copy the built frontend output from the builder stage
COPY --from=builder /webapp/out frontend/webapp/out
WORKDIR /app/frontend
ARG TARGETARCH
# Compile the Go binary for the correct architecture
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o odigos-ui

# Stage 4: Final Image
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
# Copy the compiled Go binary to the final image
COPY --from=backend /app/frontend/odigos-ui .
USER 65532:65532
ENTRYPOINT ["/app/odigos-ui"]
