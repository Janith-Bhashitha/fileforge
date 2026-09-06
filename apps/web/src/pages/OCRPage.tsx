import { useRef, useState } from 'react'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

type Status = 'idle' | 'uploading' | 'processing' | 'done' | 'error'

export function OCRPage() {
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')
  const [text, setText] = useState('')
  const [copied, setCopied] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFile(null)
    setStatus('idle')
    setError('')
    setText('')
    setCopied(false)
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
        operation: 'ocr',
        version: 'v1',
      })

      const blob = await api.downloadBlob(`/api/v1/files/${result.id}/download`)
      setText(await blob.text())
      setStatus('done')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
      setStatus('error')
    }
  }

  async function handleCopy() {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const isBusy = status === 'uploading' || status === 'processing'

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>OCR</h1>
          <p>Extract text from scanned documents and images, powered by Tesseract.</p>
        </div>
      </div>

      <div className="card">
        <div className="dropzone" onClick={() => fileInputRef.current?.click()} style={{ cursor: 'pointer' }}>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,application/pdf"
            onChange={(e) => {
              const selected = e.target.files?.[0]
              if (selected) {
                setFile(selected)
                setStatus('idle')
                setError('')
                setText('')
              }
            }}
            style={{ display: 'none' }}
          />
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">{file ? file.name : 'Click to choose an image or PDF'}</div>
          <div className="dropzone-sub">
            {file ? `${(file.size / 1024).toFixed(0)} KB` : 'Scanned PDFs are rasterized page by page automatically'}
          </div>
        </div>
      </div>

      {error && (
        <div className="form-error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      )}

      {!text && (
        <div className="card">
          <button
            className="btn-primary"
            type="button"
            style={{ width: '100%' }}
            disabled={!file || isBusy}
            onClick={handleRun}
          >
            {status === 'uploading' ? 'Uploading…' : status === 'processing' ? 'Running OCR…' : 'Extract Text'}
          </button>
        </div>
      )}

      {text && (
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
            <div className="card-title" style={{ marginBottom: 0 }}>
              Extracted Text
            </div>
            <div className="page-header-actions">
              <button className="btn-secondary" type="button" onClick={handleCopy}>
                <Icon name="copy" size={14} /> {copied ? 'Copied!' : 'Copy'}
              </button>
              <button className="btn-secondary" type="button" onClick={reset}>
                New Document
              </button>
            </div>
          </div>
          <div className="code-block" style={{ whiteSpace: 'pre-wrap', maxHeight: 500, overflowY: 'auto' }}>
            {text}
          </div>
        </div>
      )}
    </div>
  )
}
