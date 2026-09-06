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

ng external contributions — none is set yet._

