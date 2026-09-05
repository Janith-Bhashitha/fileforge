FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
# The cleanup sweep ships in the same image rather than its own: it needs the
# identical config and storage backend, and it's a few MB of static binary.
# Deployments run it as a scheduled task against this image.
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cleanup ./cmd/cleanup

FROM alpine:3.20
RUN apk add --no-cache libreoffice poppler-utils font-noto ttf-dejavu \
    && adduser -D -u 1000 appuser \
    && mkdir -p /data/storage \
    && chown -R appuser:appuser /data/storage

ENV STORAGE_DIR=/data/storage
ENV HOME=/data/storage

COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/cleanup /bin/cleanup

USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
