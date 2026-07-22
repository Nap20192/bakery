FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bakery-frontend ./frontend

FROM alpine:3.23

RUN addgroup -S bakery && adduser -S -G bakery bakery

ENV FRONTEND_ADDR=:8080
ENV BACKEND_URL=http://bakery.railway.internal:8080

COPY --from=builder /out/bakery-frontend /usr/local/bin/bakery-frontend

USER bakery
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/bakery-frontend"]
