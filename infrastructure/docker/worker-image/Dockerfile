FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker-image ./cmd/worker-image

# image-to-pdf/convert/resize are pure Go; pdf-to-image shells out to
# poppler's pdftoppm (no pure-Go PDF rasterizer exists), so that's the one
# OS package this worker needs that worker-pdf doesn't.
FROM alpine:3.20
RUN apk add --no-cache poppler-utils \
    && adduser -D -u 1000 appuser \
    && mkdir -p /data/storage \
    && chown -R appuser:appuser /data/storage

ENV STORAGE_DIR=/data/storage
ENV HOME=/data/storage

COPY --from=builder /bin/worker-image /bin/worker-image

USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/worker-image"]
