ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.19

FROM golang:1.24.1-alpine3.21 AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/knx-manager ./cmd/server

FROM ${BUILD_FROM}
WORKDIR /app

COPY --from=builder /out/knx-manager /usr/local/bin/knx-manager
COPY web ./web

ENV CUSTOM_COMPONENTS_DIR=/config/custom_components
ENV KNX_MANAGER_LISTEN=:8080
ENV KNX_MANAGER_STATIC_DIR=/app/web/static

CMD ["/usr/local/bin/knx-manager"]
