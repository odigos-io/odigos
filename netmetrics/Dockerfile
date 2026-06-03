# Build the netmetrics enrichment service.
# Multi-stage: static binary on a minimal base. Needs host /proc + pid namespace at
# runtime (run with --pid=host --privileged), and host network to scrape OBI + serve.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/netmetrics-enricher ./cmd/enricher

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/netmetrics-enricher /usr/local/bin/netmetrics-enricher
# nonroot can't read other procs' /proc/<pid>/fd; deployments run with the needed caps.
USER 0:0
ENTRYPOINT ["/usr/local/bin/netmetrics-enricher"]
