FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/radiusctl ./cmd/radiusctl

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq freeradius-utils bash iproute2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tests

COPY --from=builder /bin/radiusctl /usr/local/bin/radiusctl
COPY scripts/integration-test.sh /tests/integration-test.sh

ENV RADIUS_SERVER=http://radius-go:8083
ENV RADIUS_SECRET=change-me-in-production
ENV SERVER=http://radius-go:8083
ENV SECRET=change-me-in-production

CMD ["/tests/integration-test.sh"]
