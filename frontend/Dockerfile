# Stage 1: Dependency Installation (deps)
FROM --platform=$BUILDPLATFORM node:20.2.0 AS deps
WORKDIR /app

# Copy only the package.json and yarn.lock to leverage layer caching
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./

# Use cache mount to speed up yarn install
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn/v6 \
    yarn install --frozen-lockfile --prefer-offline

# Stage 2: Frontend Build (builder)
FROM --platform=$BUILDPLATFORM node:20.2.0 AS builder
WORKDIR /webapp

# Copy node_modules from the deps stage to avoid reinstalling dependencies
COPY --from=deps /app/node_modules ./node_modules

# Copy only the necessary source files to build the frontend
COPY frontend/webapp . 

# Build the frontend
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn/v6 \
    yarn build

# Stage 3: Backend Build (backend)
FROM --platform=$BUILDPLATFORM golang:1.22 AS backend
WORKDIR /app

# Copy only the go.mod and go.sum files first to leverage caching of dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Copy the built frontend assets from the builder stage
COPY --from=builder /webapp/out frontend/webapp/out

# Set the working directory for the Go build process
WORKDIR /app/frontend

# Build the Go binary with environment variables set to disable CGO for cross-platform compatibility
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o odigos-ui

# Final Stage: Minimal Runtime Image
FROM gcr.io/distroless/static:nonroot
WORKDIR /app

# Copy the compiled Go binary from the backend stage
COPY --from=backend /app/frontend/odigos-ui .

# Set a non-root user for security
USER 65532:65532

# Set the entrypoint to the compiled binary
ENTRYPOINT ["/app/odigos-ui"]
