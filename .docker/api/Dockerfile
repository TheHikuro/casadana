FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server

FROM scratch

# A scratch image has no CA trust store, so crypto/tls cannot verify ANY peer
# certificate. Without this the only outbound HTTPS call the API makes -- Resend --
# fails with:
#   tls: failed to verify certificate: x509: certificate signed by unknown authority
# regardless of whether RESEND_API_KEY is valid, so every booking confirmation and
# admin notification is silently dropped (logged at WARN, booking still succeeds).
# Observed on demo.casa-dana.com 2026-08-10.
#
# tzdata is deliberately NOT copied: the API uses UTC only (no time.LoadLocation or
# time.Local anywhere), and Go falls back to UTC when the zoneinfo database is absent.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /app/bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
