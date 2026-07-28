FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
# go.sum will be created on first build if not present
COPY go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /bin/bwtsearch ./cmd/bwtsearch

# ── runtime image ───────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

WORKDIR /data
COPY --from=builder /bin/bwtsearch /usr/local/bin/bwtsearch
COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

ENTRYPOINT ["/usr/local/bin/bwtsearch"]
CMD ["--help"]
