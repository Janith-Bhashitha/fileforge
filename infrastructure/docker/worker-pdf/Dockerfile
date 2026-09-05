FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker-pdf ./cmd/worker-pdf

# pdf-merge/split/compress are all pure Go (pdfcpu) - no OS packages needed
# beyond the base image, unlike worker-image and worker-office.
FROM alpine:3.20
RUN adduser -D -u 1000 appuser \
    && mkdir -p /data/storage \
    && chown -R appuser:appuser /data/storage

ENV STORAGE_DIR=/data/storage
ENV HOME=/data/storage

COPY --from=builder /bin/worker-pdf /bin/worker-pdf

USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/worker-pdf"]
