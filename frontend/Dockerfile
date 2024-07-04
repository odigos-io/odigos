FROM node:20.2.0 AS deps
WORKDIR /app
COPY frontend/webapp/package.json frontend/webapp/yarn.lock ./
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn \
    yarn install --frozen-lockfile

FROM node:20.2.0 AS builder
WORKDIR /webapp
COPY --from=deps /app/node_modules ./node_modules
COPY frontend//webapp .
RUN yarn build


FROM golang:1.22 AS backend
WORKDIR /app
COPY . .
COPY --from=builder /webapp/out frontend/webapp/out
WORKDIR /app/frontend
RUN CGO_ENABLED=0 go build -o odigos-ui

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=backend /app/frontend/odigos-ui .
USER 65532:65532
ENTRYPOINT ["/app/odigos-ui"]