FROM golang:1.20.4 AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o init

FROM busybox
WORKDIR /odigos
COPY --from=build /app/init /odigos-init/init
RUN chmod -R go+r /odigos-init