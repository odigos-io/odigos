# syntax=docker/dockerfile:1.7

FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mcp ./

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mcp /usr/local/bin/mcp
EXPOSE 9090
ENV MCP_CLUSTER_LISTEN_ADDR=0.0.0.0:9090
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/mcp"]
