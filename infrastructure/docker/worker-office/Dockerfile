FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker-office ./cmd/worker-office

# docx/pptx/xlsx-to-pdf shell out to LibreOffice; txt-to-pdf is pure Go
# (fpdf) and needs nothing extra.
FROM alpine:3.20
RUN apk add --no-cache libreoffice font-noto ttf-dejavu \
    && adduser -D -u 1000 appuser \
    && mkdir -p /data/storage \
    && chown -R appuser:appuser /data/storage

ENV STORAGE_DIR=/data/storage
ENV HOME=/data/storage

COPY --from=builder /bin/worker-office /bin/worker-office

USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/worker-office"]
