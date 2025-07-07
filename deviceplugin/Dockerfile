# -------- Stage 1: Build --------
FROM golang:1.24 AS builder

WORKDIR /build

# Copy shared monorepo dependencies
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY procdiscovery/ procdiscovery/
COPY opampserver/ opampserver/
COPY instrumentation/ instrumentation/

# Copy the deviceplugin module directory
COPY deviceplugin/ deviceplugin/

# Set workdir to deviceplugin module
WORKDIR /build/deviceplugin

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /deviceplugin ./cmd

# -------- Stage 2: Distroless minimal runtime --------
FROM gcr.io/distroless/base:latest

# Copy the binary from the builder stage
COPY --from=builder /deviceplugin /deviceplugin

# Entrypoint
ENTRYPOINT ["/deviceplugin"]
