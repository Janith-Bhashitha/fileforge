import { useRef, useState } from 'react'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

interface Analysis {
  summary: string
  category: string
  tags: string[]
  ocr_used: boolean
}

type Status = 'idle' | 'uploading' | 'processing' | 'done' | 'error' | 'not-configured'

const categoryColors: Record<string, string> = {
  invoice: 'purple',
  contract: 'blue',
  letter: 'teal',
  report: 'green',
  resume: 'orange',
  receipt: 'yellow',
  form: 'gray',
  article: 'red',
}

export function AIProcessingPage() {
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')
  const [analysis, setAnalysis] = useState<Analysis | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFile(null)
    setStatus('idle')
    setError('')
    setAnalysis(null)
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
        operation: 'ai-analyze',
        version: 'v1',
      })

      const blob = await api.downloadBlob(`/api/v1/files/${result.id}/download`)
      setAnalysis(JSON.parse(await blob.text()))
      setStatus('done')
    } catch (err) {
      if (err instanceof ApiError && err.message.includes('not configured')) {
        setStatus('not-configured')
        return
      }
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
      setStatus('error')
    }
  }

  const isBusy = status === 'uploading' || status === 'processing'

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>AI Processing</h1>
          <p>Summarize, classify and tag documents using Gemini.</p>
        </div>
      </div>

      <div className="card">
        <div className="dropzone" onClick={() => fileInputRef.current?.click()} style={{ cursor: 'pointer' }}>
          <input
            ref={fileInputRef}
            type="file"
            accept=".txt,.md,application/pdf,image/*"
            onChange={(e) => {
              const selected = e.target.files?.[0]
              if (selected) {
                setFile(selected)
                setStatus('idle')
                setError('')
                setAnalysis(null)
              }
            }}
            style={{ display: 'none' }}
          />
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">{file ? file.name : 'Click to choose a document'}</div>
          <div className="dropzone-sub">
            {file ? `${(file.size / 1024).toFixed(0)} KB` : 'PDF, TXT, Markdown, or a scanned image'}
          </div>
        </div>
      </div>

      {status === 'not-configured' && (
        <div className="banner banner-warning">
          AI Processing needs a free Gemini API key to be set on the server (GEMINI_API_KEY). Get one at{' '}
          <a href="https://aistudio.google.com/apikey" target="_blank" rel="noreferrer">
            aistudio.google.com
          </a>{' '}
          — no credit card required.
        </div>
      )}

      {error && (
        <div className="form-error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      )}

      {!analysis && status !== 'not-configured' && (
        <div className="card">
          <button
            className="btn-primary"
            type="button"
            style={{ width: '100%' }}
            disabled={!file || isBusy}
            onClick={handleRun}
          >
            {status === 'uploading' ? 'Uploading…' : status === 'processing' ? 'Analyzing with Gemini…' : 'Analyze'}
          </button>
        </div>
      )}

      {analysis && (
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
            <div className="card-title" style={{ marginBottom: 0 }}>
              Analysis
            </div>
            <button className="btn-secondary" type="button" onClick={reset}>
              New Document
            </button>
          </div>

          {analysis.ocr_used && (
            <div className="banner banner-info">This document had no text layer, so it was read via OCR first.</div>
          )}

          <div style={{ marginBottom: 16 }}>
            <span className={`file-type-icon file-type-icon-${categoryColors[analysis.category] ?? 'gray'}`}>
              {analysis.category.slice(0, 3).toUpperCase()}
            </span>
            <span style={{ marginLeft: 10, fontWeight: 600, textTransform: 'capitalize' }}>{analysis.category}</span>
          </div>

          <p style={{ fontSize: 14, lineHeight: 1.6, marginBottom: 16 }}>{analysis.summary}</p>

          <div className="pill-group">
            {analysis.tags.map((tag) => (
              <span className="pill-option" key={tag} style={{ cursor: 'default' }}>
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
