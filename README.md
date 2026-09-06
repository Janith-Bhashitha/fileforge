# FileForge

AI-powered, all-in-one file conversion & processing platform.
# FileForge

[![CI](https://github.com/Janith-Bhashitha/fileforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Janith-Bhashitha/fileforge/actions/workflows/ci.yml)

An all-in-one file conversion and processing platform — convert, batch process, extract intelligence from, and edit documents, all behind a public REST API.

## Features

**Conversion & processing** — 21 operations across PDF, Office, image and text formats, run through a pluggable processor registry:

| Category | Operations |
|---|---|
| To PDF | image → PDF, DOCX/PPTX/XLSX → PDF, TXT → PDF |
| From PDF | PDF → JPG/PNG |
| Organize | merge, split, rotate, remove pages, extract pages |
| Optimize & secure | compress, watermark, password-protect, unlock |
| Edit & sign | overlay text, images and signatures onto an existing page |
| Image tools | convert format, resize |
| AI & documents | OCR (local, Tesseract), document insights, AI summarize/classify (Gemini) |

**Batch processing** — queue many files against one operation, with per-item retry and progress tracking, backed by a Redis job queue and 4 independently scalable worker services.

**Edit & Sign** — a canvas-based PDF editor (pdf.js) for placing typed text and a drawn signature onto a page. Overlay editing, not content editing: the original page is never altered underneath.

**Developer API** — everything above is available over `/api/v1`, authenticated by JWT or an API key:
- API keys are bcrypt-hashed and shown once at creation
- Webhooks deliver HMAC-SHA256-signed payloads with automatic retry and a delivery log, so a receiver can verify a request actually came from FileForge
- Per-user rate limiting and structured audit logging on every state-changing request

**Storage** — pluggable local-disk or S3 backend (config-only swap), with presigned direct-to-S3 uploads so file bytes never pass through the API server. Every object is namespaced per owner.

## Tech stack

**Backend** — Go, [chi](https://github.com/go-chi/chi) router, PostgreSQL via [pgx](https://github.com/jackc/pgx), Redis-backed job queue, JWT + API-key auth, [pdfcpu](https://github.com/pdfcpu/pdfcpu)/[fpdf](https://github.com/go-pdf/fpdf) for PDF operations, LibreOffice (headless) for Office conversions, Tesseract for OCR, Prometheus metrics.

**Frontend** — React 19, TypeScript, [TanStack Query](https://tanstack.com/query), React Router, Vite, [pdfjs-dist](https://mozilla.github.io/pdf.js/) (code-split so it only loads on the Edit & Sign page).

**Infrastructure** — Docker Compose (6 services: API + 4 workers + web/nginx), GitHub Actions CI/CD (build, test, and publish all 6 images on every merge to `main`), Terraform-managed AWS deployment (EC2, S3, IAM), Prometheus + Grafana for observability.

## Architecture

```
                    ┌─────────┐
   Browser ────────▶│  nginx  │──── static assets (React SPA)
                    └────┬────┘
                         │ /api/*
                    ┌────▼────┐        ┌──────────┐
                    │   API   │───────▶│ Postgres │
                    └────┬────┘        └──────────┘
                         │
              ┌──────────┼──────────────────┬─────────────┐
              ▼          ▼                  ▼             ▼
       ┌───────────┐ ┌────────┐      ┌────────────┐ ┌───────────┐
       │ worker-pdf│ │worker- │      │ worker-ai  │ │worker-    │
       │           │ │image   │      │ (OCR, LLM) │ │office     │
       └─────┬─────┘ └───┬────┘      └─────┬──────┘ └─────┬─────┘
             │           │                 │              │
             └───────────┴────────Redis────┴──────────────┘
                                   (job queue)

       Storage: local disk (dev) or S3 (presigned direct upload)
```

Every conversion operation registers into a single `Registry` shared by the synchronous `/convert` endpoint and every async worker — adding a new operation never touches the queue, the API routes, or the other workers.

## Getting started

Requires Docker and Docker Compose.

```bash
git clone https://github.com/Janith-Bhashitha/fileforge.git
cd fileforge
cp .env.example .env
docker compose up -d
```

Apply migrations (needs [golang-migrate](https://github.com/golang-migrate/migrate)):

```bash
migrate -path services/api/migrations \
  -database "postgres://fileforge:fileforge@localhost:5433/fileforge?sslmode=disable" up
```

The app is served at `http://localhost:3001`, the API directly at `http://localhost:8080`.

### Configuration

All variables are documented in [`.env.example`](.env.example). Nothing is required to run the core platform — everything below is additive:

| Variable | Purpose | If unset |
|---|---|---|
| `GEMINI_API_KEY` | AI summarize/classify ([free key](https://aistudio.google.com/apikey), no card) | `ai-analyze` fails clearly per-request; OCR and Document Insights are unaffected |
| `SMTP_*` | Password-reset email | Reset still works — the API logs the reset link instead |
| `RATE_LIMIT_PER_MINUTE`, `MAX_CONCURRENT_JOBS`, `RETENTION_DAYS` | Hardening knobs | Sensible defaults apply |
| `DOCKERHUB_USERNAME` | Where CI publishes images | Only needed for a real deploy, not local dev |

### Object storage

Local disk by default. To run against S3-compatible storage locally (via MinIO, no AWS account needed):

```bash
docker compose --profile s3 up -d
```

### Observability

```bash
docker compose --profile observability up -d
```

Grafana at `http://localhost:3000` (anonymous access, admin dashboards pre-provisioned), Prometheus at `http://localhost:9090`.

## Deployment

[`infrastructure/terraform-ec2/`](infrastructure/terraform-ec2/) provisions a complete free-tier-eligible deployment on AWS: a single EC2 instance running the full Docker Compose stack, an S3 bucket for file storage, an IAM role scoped to that one bucket, a security group with SSH closed by default (SSM Session Manager is used for shell access instead), and a billing alarm. See the [deployment README](infrastructure/terraform-ec2/README.md) for the full runbook.

## API

Every endpoint under `/api/v1` accepts either a JWT (from `/api/auth/login`) or an `X-API-Key` header, so the same surface serves the web app and third-party integrations:

```bash
curl -X POST https://your-host/api/v1/convert \
  -H "X-API-Key: ffk_..." \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": "...",
    "operation": "pdf-compress",
    "version": "v1"
  }'
```

Manage API keys and webhooks from the app's Developer section, or via `/api/v1/api-keys` and `/api/v1/webhooks` directly.

## Project structure

```
apps/web/                  React + TypeScript frontend
services/api/
  cmd/                      6 binaries: api, worker-pdf, worker-image,
                            worker-office, worker-ai, cleanup
  internal/
    convert/                Processor registry + implementations
      pdfops/ imageops/ office/ txtops/ aiops/
    handlers/               HTTP handlers
    storage/                Local disk / S3 abstraction
    webhooks/ apikeys/      Developer API
    workerconsumer/         Shared async job-processing engine
  migrations/               golang-migrate SQL migrations
infrastructure/
  docker/                   Per-service Dockerfiles, nginx config
  terraform-ec2/            AWS deployment
.github/workflows/ci.yml   Build, test, publish (6 images) on every merge
```

## License

_Add a license before accepting external contributions — none is set yet._

