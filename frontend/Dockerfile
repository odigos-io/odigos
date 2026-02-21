FROM --platform=$BUILDPLATFORM node:20.9.0 AS deps
WORKDIR /app
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./
RUN yarn install --frozen-lockfile

FROM --platform=$BUILDPLATFORM node:20.9.0 AS builder
WORKDIR /webapp
COPY --from=deps /app/node_modules ./node_modules
COPY frontend/webapp .
RUN yarn build

FROM --platform=$BUILDPLATFORM golang:1.26 AS backend
WORKDIR /app
COPY . .
RUN rm -rf frontend/webapp/*
COPY --from=builder /webapp/out frontend/webapp/out
WORKDIR /app/frontend
ARG TARGETARCH
ARG LD_FLAGS
ARG RHEL=false
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="${LD_FLAGS}" -o odigos-ui
RUN if [ "$RHEL" = "true" ] ; then \
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
