import { useRef, useState } from 'react'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

interface PDFMetadata {
  page_count: number
  title?: string
  author?: string
  subject?: string
  creator?: string
  creation_date?: string
}

interface Insights {
  word_count: number
  character_count: number
  sentence_count: number
  language: string
  language_name: string
  language_confident: boolean
  ocr_used: boolean
  top_keywords: string[]
  pdf_metadata?: PDFMetadata
  text_preview: string
}

type Status = 'idle' | 'uploading' | 'processing' | 'done' | 'error'

export function DocumentInsightsPage() {
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')
  const [insights, setInsights] = useState<Insights | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFile(null)
    setStatus('idle')
    setError('')
    setInsights(null)
  }

  async function handleRun() {
    if (!file) return
    setStatus('uploading')
    setError('')

    try {
      const formData = new FormData()
      formData.append('file', file)
      const uploaded = await api.upload<FileResponse>('/api/v1/files', formData)

      setStatus('processing')
      const result = await api.post<FileResponse>('/api/v1/convert', {
        file_id: uploaded.id,
        operation: 'document-insights',
        version: 'v1',
      })

      const blob = await api.downloadBlob(`/api/v1/files/${result.id}/download`)
      setInsights(JSON.parse(await blob.text()))
      setStatus('done')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
      setStatus('error')
    }
  }

  const isBusy = status === 'uploading' || status === 'processing'

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Document Insights</h1>
          <p>Word counts, language detection, keywords and metadata — computed directly, no model involved.</p>
        </div>
      </div>

      <div className="card">
        <div className="dropzone" onClick={() => fileInputRef.current?.click()} style={{ cursor: 'pointer' }}>
          <input
            ref={fileInputRef}
            type="file"
            accept=".txt,.md,.csv,application/pdf,image/*"
            onChange={(e) => {
              const selected = e.target.files?.[0]
              if (selected) {
                setFile(selected)
                setStatus('idle')
                setError('')
                setInsights(null)
              }
            }}
            style={{ display: 'none' }}
          />
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">{file ? file.name : 'Click to choose a document'}</div>
          <div className="dropzone-sub">
            {file ? `${(file.size / 1024).toFixed(0)} KB` : 'PDF, TXT, Markdown, CSV, or an image'}
          </div>
        </div>
      </div>

      {error && (
        <div className="form-error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      )}

      {!insights && (
        <div className="card">
          <button
            className="btn-primary"
            type="button"
            style={{ width: '100%' }}
            disabled={!file || isBusy}
            onClick={handleRun}
          >
            {status === 'uploading' ? 'Uploading…' : status === 'processing' ? 'Analyzing…' : 'Analyze Document'}
          </button>
        </div>
      )}

      {insights && (
        <>
          <div className="card">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <div className="card-title" style={{ marginBottom: 0 }}>
                Analysis
              </div>
              <button className="btn-secondary" type="button" onClick={reset}>
                New Document
              </button>
            </div>

            {insights.ocr_used && (
              <div className="banner banner-info">
                No text layer was found, so this document was read via OCR — accuracy depends on scan quality.
              </div>
            )}

            <div className="stat-mini-grid">
              <div className="stat-mini">
                <div className="stat-mini-value">{insights.word_count}</div>
                <div className="stat-mini-label">Words</div>
              </div>
              <div className="stat-mini">
                <div className="stat-mini-value">{insights.sentence_count}</div>
                <div className="stat-mini-label">Sentences</div>
              </div>
              <div className="stat-mini">
                <div className="stat-mini-value">{insights.character_count}</div>
                <div className="stat-mini-label">Characters</div>
              </div>
              <div className="stat-mini">
                <div className="stat-mini-value">{insights.language_name}</div>
                <div className="stat-mini-label">
                  Language{!insights.language_confident && ' (uncertain)'}
                </div>
              </div>
            </div>

            {insights.top_keywords.length > 0 && (
              <>
                <div className="card-title" style={{ marginTop: 8 }}>
                  Top Keywords
                </div>
                <div className="pill-group">
                  {insights.top_keywords.map((k) => (
                    <span className="pill-option" key={k} style={{ cursor: 'default' }}>
                      {k}
                    </span>
                  ))}
                </div>
              </>
            )}
          </div>

          {insights.pdf_metadata && (
            <div className="card">
              <div className="card-title">PDF Metadata</div>
              <div className="service-row">
                <span className="service-name">Pages</span>
                <span>{insights.pdf_metadata.page_count}</span>
              </div>
              {insights.pdf_metadata.title && (
                <div className="service-row">
                  <span className="service-name">Title</span>
                  <span>{insights.pdf_metadata.title}</span>
                </div>
              )}
              {insights.pdf_metadata.author && (
                <div className="service-row">
                  <span className="service-name">Author</span>
                  <span>{insights.pdf_metadata.author}</span>
                </div>
              )}
              {insights.pdf_metadata.creator && (
                <div className="service-row">
                  <span className="service-name">Creator</span>
                  <span>{insights.pdf_metadata.creator}</span>
                </div>
              )}
              {insights.pdf_metadata.creation_date && (
                <div className="service-row">
                  <span className="service-name">Created</span>
                  <span>{insights.pdf_metadata.creation_date}</span>
                </div>
              )}
            </div>
          )}

          <div className="card">
            <div className="card-title">Text Preview</div>
            <p style={{ fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>
              {insights.text_preview}
            </p>
          </div>
        </>
      )}
    </div>
  )
}
