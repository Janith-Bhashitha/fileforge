import { useRef, useState, type ChangeEvent } from 'react'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

interface OperationConfig {
  label: string
  operation: string
  options?: Record<string, string>
  accept: string
  implemented: boolean
}

const operations: OperationConfig[] = [
  { label: 'JPG → PDF', operation: 'image-to-pdf', accept: 'image/jpeg', implemented: true },
  { label: 'PNG → PDF', operation: 'image-to-pdf', accept: 'image/png', implemented: true },
  { label: 'PDF → JPG', operation: 'pdf-to-image', options: { format: 'jpeg' }, accept: 'application/pdf', implemented: true },
  { label: 'PDF → PNG', operation: 'pdf-to-image', options: { format: 'png' }, accept: 'application/pdf', implemented: true },
  { label: 'DOCX → PDF', operation: 'docx-to-pdf', accept: '.docx', implemented: true },
  { label: 'Compress PDF', operation: 'pdf-compress', accept: 'application/pdf', implemented: true },
  { label: 'Split PDF', operation: 'pdf-split', accept: 'application/pdf', implemented: true },
  { label: 'PPTX → PDF', operation: 'pptx-to-pdf', accept: '.pptx', implemented: true },
  { label: 'XLSX → PDF', operation: 'xlsx-to-pdf', accept: '.xlsx', implemented: true },
  { label: 'TXT → PDF', operation: 'txt-to-pdf', accept: '.txt', implemented: true },
  { label: 'JPG → PNG', operation: 'image-convert', options: { format: 'png' }, accept: 'image/jpeg', implemented: true },
  { label: 'PNG → JPG', operation: 'image-convert', options: { format: 'jpeg' }, accept: 'image/png', implemented: true },
  { label: 'Resize Image', operation: 'image-resize', options: { max_width: '1200' }, accept: 'image/*', implemented: true },
]

type Status = 'idle' | 'uploading' | 'converting' | 'done' | 'error'

export function ConvertPage() {
  const [selectedOp, setSelectedOp] = useState(operations[0])
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')
  const [result, setResult] = useState<FileResponse | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFile(null)
    setStatus('idle')
    setError('')
    setResult(null)
  }

  function handleSelectOperation(op: OperationConfig) {
    setSelectedOp(op)
    reset()
  }

  function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0]
    if (selected) {
      setFile(selected)
      setStatus('idle')
      setError('')
      setResult(null)
    }
  }

  async function handleConvert() {
    if (!file || !selectedOp.implemented) return
    setStatus('uploading')
    setError('')

    try {
      const formData = new FormData()
      formData.append('file', file)
      const uploaded = await api.upload<FileResponse>('/api/v1/files', formData)

      setStatus('converting')
      const converted = await api.post<FileResponse>('/api/v1/convert', {
        file_id: uploaded.id,
        operation: selectedOp.operation,
        version: 'v1',
        options: selectedOp.options ?? {},
      })

      setResult(converted)
      setStatus('done')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
      setStatus('error')
    }
  }

  async function handleDownload() {
    if (!result) return
    const blob = await api.downloadBlob(`/api/v1/files/${result.id}/download`)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = result.filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  const isBusy = status === 'uploading' || status === 'converting'

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Convert Files</h1>
          <p>Transform your files into the format you need.</p>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Conversion Type</div>
        <div className="pill-group">
          {operations.map((op) => (
            <button
              key={op.label}
              type="button"
              className={`pill-option ${selectedOp.label === op.label ? 'pill-option-active' : ''} ${
                !op.implemented ? 'pill-option-soon' : ''
              }`}
              onClick={() => handleSelectOperation(op)}
            >
              {op.label}
              {!op.implemented && <span className="pill-option-soon-badge">Soon</span>}
            </button>
          ))}
        </div>
      </div>

      <div className="card">
        <div className="dropzone" onClick={() => fileInputRef.current?.click()} style={{ cursor: 'pointer' }}>
          <input
            ref={fileInputRef}
            type="file"
            accept={selectedOp.accept}
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">{file ? file.name : 'Click to choose a file'}</div>
          <div className="dropzone-sub">
            {file ? `${(file.size / 1024).toFixed(0)} KB` : 'or browse from your device'}
          </div>
        </div>
      </div>

      {error && (
        <div className="form-error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      )}

      {!selectedOp.implemented && (
        <div className="banner banner-info">
          {selectedOp.label} isn't built yet — it's on the roadmap and will work here once it ships.
        </div>
      )}

      <div className="card">
        {status === 'done' && result ? (
          <button className="btn-primary" type="button" style={{ width: '100%' }} onClick={handleDownload}>
            <Icon name="download" size={14} /> Download {result.filename}
          </button>
        ) : (
          <button
            className="btn-primary"
            type="button"
            disabled={!file || isBusy || !selectedOp.implemented}
            style={{ width: '100%' }}
            onClick={handleConvert}
          >
            {!selectedOp.implemented
              ? 'Coming soon'
              : status === 'uploading'
                ? 'Uploading…'
                : status === 'converting'
                  ? 'Converting…'
                  : 'Start Conversion'}
          </button>
        )}
      </div>
    </div>
  )
}
