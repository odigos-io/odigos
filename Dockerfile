FROM golang:1.18 as builder
ARG SERVICE_NAME
WORKDIR /workspace
# Copy the go source
COPY . .
# Build
WORKDIR /workspace/$SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ../app main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/app .
ENTRYPOINT ["/app"]
