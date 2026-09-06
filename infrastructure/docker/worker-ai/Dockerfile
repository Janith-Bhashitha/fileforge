FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker-ai ./cmd/worker-ai

# ocr and document-insights shell out to tesseract (text recognition) and
# poppler's pdftotext/pdftoppm (text-layer extraction and, when that's
# absent, rasterising a scanned PDF's pages for tesseract to read).
# ai-analyze needs neither - it's a plain HTTP call - but reuses the same
# text-extraction path as document-insights, so it needs them too.
FROM alpine:3.20
RUN apk add --no-cache tesseract-ocr tesseract-ocr-data-eng poppler-utils \
    && adduser -D -u 1000 appuser \
    && mkdir -p /data/storage \
    && chown -R appuser:appuser /data/storage

ENV STORAGE_DIR=/data/storage
ENV HOME=/data/storage

COPY --from=builder /bin/worker-ai /bin/worker-ai

USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/worker-ai"]
