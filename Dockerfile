ARG GO_IMAGE=golang:1.26-alpine
ARG RUNTIME_IMAGE=alpine:3.23

FROM ${GO_IMAGE} AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/bakery ./cmd/bakery

FROM ${RUNTIME_IMAGE}

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app && \
    mkdir /data && chown app:app /data

WORKDIR /app

COPY --from=builder /out/bakery .
COPY migrations ./migrations
COPY templates ./templates

ENV LOG_PRETTY=false \
    LOG_LEVEL=INFO \
    MIGRATIONS_DIR=/app/migrations \
    SQLITE_PATH=/data/bakery.db

USER app
EXPOSE 8080

ENTRYPOINT ["/app/bakery"]
