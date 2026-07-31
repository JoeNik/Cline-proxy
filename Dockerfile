FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o cline-proxy .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/cline-proxy .

EXPOSE 3457

VOLUME ["/app/data"]

ENV HOST=0.0.0.0
ENV PORT=3457

ENTRYPOINT ["/app/cline-proxy"]
