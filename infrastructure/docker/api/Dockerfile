FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 1000 appuser
COPY --from=builder /bin/api /bin/api
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
