ARG GO_IMAGE=golang:alpine
ARG RUNTIME_IMAGE=alpine:latest

FROM ${GO_IMAGE} AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags="-s -w" -o /out/bakery ./cmd/worker

FROM ${RUNTIME_IMAGE}

RUN apk add --no-cache ca-certificates tzdata && \
	addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/bakery /app/bakery
COPY migrations /app/migrations

ENV LOG_PRETTY=false \
	LOG_LEVEL=INFO \
	MIGRATIONS_DIR=/app/migrations

USER app

ENTRYPOINT ["/app/bakery"]
