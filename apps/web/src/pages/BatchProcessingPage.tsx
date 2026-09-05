import { useMemo, useRef, useState, type ChangeEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { StatusBadge, type Status } from '../components/StatusBadge'
import { api, ApiError } from '../lib/api'
import { formatDate, mimeTypeToLabel } from '../lib/format'
import type { FileRecord } from '../lib/types'

// A prompt is a value only the user can supply (watermark text, password).
// In a batch it applies to every file, the same way the operation does.
interface OptionPrompt {
  key: string
  label: string
  placeholder: string
  type?: 'text' | 'password'
}

interface OperationConfig {
  label: string
  operation: string
  options?: Record<string, string>
  accept: string
  prompts?: OptionPrompt[]
}

const operations: OperationConfig[] = [
  { label: 'JPG/PNG → PDF', operation: 'image-to-pdf', accept: 'image/jpeg,image/png' },
  { label: 'PDF → JPG', operation: 'pdf-to-image', options: { format: 'jpeg' }, accept: 'application/pdf' },
  { label: 'PDF → PNG', operation: 'pdf-to-image', options: { format: 'png' }, accept: 'application/pdf' },
  { label: 'DOCX → PDF', operation: 'docx-to-pdf', accept: '.docx' },
  { label: 'PPTX → PDF', operation: 'pptx-to-pdf', accept: '.pptx' },
  { label: 'XLSX → PDF', operation: 'xlsx-to-pdf', accept: '.xlsx' },
  { label: 'TXT → PDF', operation: 'txt-to-pdf', accept: '.txt' },
  { label: 'Compress PDF', operation: 'pdf-compress', accept: 'application/pdf' },
  { label: 'Split PDF', operation: 'pdf-split', accept: 'application/pdf' },
  { label: 'JPG → PNG', operation: 'image-convert', options: { format: 'png' }, accept: 'image/jpeg' },
  { label: 'PNG → JPG', operation: 'image-convert', options: { format: 'jpeg' }, accept: 'image/png' },
  { label: 'Resize Image', operation: 'image-resize', options: { max_width: '1200' }, accept: 'image/*' },
  { label: 'Rotate PDF', operation: 'pdf-rotate', options: { angle: '90' }, accept: 'application/pdf' },
  {
    label: 'Watermark PDF',
    operation: 'pdf-watermark',
    accept: 'application/pdf',
    prompts: [{ key: 'text', label: 'Watermark text', placeholder: 'CONFIDENTIAL' }],
  },
  {
    label: 'Protect PDF',
    operation: 'pdf-protect',
    accept: 'application/pdf',
    prompts: [{ key: 'password', label: 'Password', placeholder: 'Choose a password', type: 'password' }],
  },
  {
    label: 'Unlock PDF',
    operation: 'pdf-unlock',
    accept: 'application/pdf',
    prompts: [{ key: 'password', label: 'Current password', placeholder: 'Enter the PDF password', type: 'password' }],
  },
]

interface BatchResponse {
  id: string
  operation: string
  total: number
  completed: number
  failed: number
  status: string
  created_at: string
}

interface JobItemResponse {
  id: string
  input_file_id: string
  output_file_id?: string
  status: string
  attempts: number
  last_error?: string
}

const ACTIVE_BATCH_KEY = 'active_batch_id'

function isSettled(status?: string) {
  return status === 'completed' || status === 'failed' || status === 'partially_completed'
}

export function BatchProcessingPage() {
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [selectedOp, setSelectedOp] = useState(operations[0])
  const [promptValues, setPromptValues] = useState<Record<string, string>>({})
  const [pendingFiles, setPendingFiles] = useState<File[]>([])
  const [batchId, setBatchId] = useState<string | null>(() => localStorage.getItem(ACTIVE_BATCH_KEY))
  const [createError, setCreateError] = useState('')

  const batchQuery = useQuery({
    queryKey: ['batch', batchId],
    queryFn: () => api.get<BatchResponse>(`/api/v1/batches/${batchId}`),
    enabled: !!batchId,
    refetchInterval: (query) => (isSettled(query.state.data?.status) ? false : 2000),
  })

  const itemsQuery = useQuery({
    queryKey: ['batch-items', batchId],
    queryFn: () => api.get<JobItemResponse[]>(`/api/v1/batches/${batchId}/items`),
    enabled: !!batchId,
    refetchInterval: () => (isSettled(batchQuery.data?.status) ? false : 2000),
  })

  const filesQuery = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get<FileRecord[]>('/api/v1/files'),
    enabled: !!batchId,
    refetchInterval: () => (isSettled(batchQuery.data?.status) ? false : 2000),
  })

  const filesMap = useMemo(() => {
    const map = new Map<string, FileRecord>()
    for (const f of filesQuery.data ?? []) map.set(f.id, f)
    return map
  }, [filesQuery.data])

  const createMutation = useMutation({
    mutationFn: async () => {
      const formData = new FormData()
      formData.append('operation', selectedOp.operation)
      formData.append('version', 'v1')
      const options = { ...(selectedOp.options ?? {}), ...promptValues }
      if (Object.keys(options).length > 0) formData.append('options', JSON.stringify(options))
      for (const f of pendingFiles) formData.append('files', f)
      return api.upload<BatchResponse>('/api/v1/batches', formData)
    },
    onSuccess: (batch) => {
      localStorage.setItem(ACTIVE_BATCH_KEY, batch.id)
      setBatchId(batch.id)
      setPendingFiles([])
      queryClient.setQueryData(['batch', batch.id], batch)
    },
    onError: (err) => setCreateError(err instanceof ApiError ? err.message : 'Failed to create batch'),
  })

  const retryMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/batches/${batchId}/retry-failed`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batch', batchId] })
      queryClient.invalidateQueries({ queryKey: ['batch-items', batchId] })
    },
  })

  function handleFilesSelected(e: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files ?? [])
    setPendingFiles((prev) => [...prev, ...selected])
    setCreateError('')
    e.target.value = ''
  }

  function removePendingFile(index: number) {
    setPendingFiles((prev) => prev.filter((_, i) => i !== index))
  }

  function handleSelectOperation(op: OperationConfig) {
    setSelectedOp(op)
    setPromptValues({})
    setCreateError('')
  }

  function startNewBatch() {
    localStorage.removeItem(ACTIVE_BATCH_KEY)
    setBatchId(null)
    setPendingFiles([])
    setCreateError('')
  }

  async function downloadZip() {
    if (!batchId) return
    const blob = await api.downloadBlob(`/api/v1/batches/${batchId}/download`)
    triggerDownload(blob, `batch-${batchId}.zip`)
  }

  async function downloadItemOutput(item: JobItemResponse) {
    if (!item.output_file_id) return
    const outputFile = filesMap.get(item.output_file_id)
    const blob = await api.downloadBlob(`/api/v1/files/${item.output_file_id}/download`)
    triggerDownload(blob, outputFile?.filename ?? 'output')
  }

  function triggerDownload(blob: Blob, filename: string) {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  const batch = batchQuery.data
  const items = itemsQuery.data ?? []
  const inProgress = batch ? Math.max(batch.total - batch.completed - batch.failed, 0) : 0
  const percent = batch && batch.total > 0 ? Math.round((batch.completed / batch.total) * 100) : 0
  const opLabel = operations.find((o) => o.operation === batch?.operation)?.label ?? batch?.operation

  if (!batchId || !batch) {
    return (
      <div>
        <div className="page-header">
          <div>
            <h1>Batch Processing</h1>
            <p>Process many files with one operation.</p>
          </div>
        </div>

        <div className="card">
          <div className="card-title">Conversion Type</div>
          <div className="pill-group">
            {operations.map((op) => (
              <button
                key={op.label}
                type="button"
                className={`pill-option ${selectedOp.label === op.label ? 'pill-option-active' : ''}`}
                onClick={() => handleSelectOperation(op)}
              >
                {op.label}
              </button>
            ))}
          </div>
        </div>

        {selectedOp.prompts && selectedOp.prompts.length > 0 && (
          <div className="card">
            {selectedOp.prompts.map((prompt) => (
              <div className="field-row" key={prompt.key}>
                <label htmlFor={`batch-opt-${prompt.key}`}>
                  {prompt.label} <span style={{ color: 'var(--text-muted)' }}>· applies to every file</span>
                </label>
                <input
                  id={`batch-opt-${prompt.key}`}
                  type={prompt.type === 'password' ? 'password' : 'text'}
                  placeholder={prompt.placeholder}
                  value={promptValues[prompt.key] ?? ''}
                  onChange={(e) => setPromptValues((prev) => ({ ...prev, [prompt.key]: e.target.value }))}
                />
              </div>
            ))}
          </div>
        )}

        <div className="card">
          <div
            className="dropzone"
            onClick={() => fileInputRef.current?.click()}
            style={{ cursor: 'pointer' }}
          >
            <input
              ref={fileInputRef}
              type="file"
              multiple
              accept={selectedOp.accept}
              onChange={handleFilesSelected}
              style={{ display: 'none' }}
            />
            <div className="dropzone-icon">
              <Icon name="upload" size={20} />
            </div>
            <div className="dropzone-title">Click to choose files</div>
            <div className="dropzone-sub">Select as many files as you want to process with {selectedOp.label}</div>
          </div>

          {pendingFiles.length > 0 && (
            <div style={{ marginTop: 16 }}>
              {pendingFiles.map((f, i) => (
                <div
                  key={`${f.name}-${i}`}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 0',
                    borderBottom: '1px solid var(--border)',
                  }}
                >
                  <div className="table-file-cell">
                    <FileTypeIcon type={f.name.split('.').pop() ?? 'file'} />
                    <span className="table-file-name">{f.name}</span>
                    <span className="table-file-sub">{(f.size / 1024).toFixed(0)} KB</span>
                  </div>
                  <button type="button" className="btn-danger-ghost" onClick={() => removePendingFile(i)}>
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {createError && (
          <div className="form-error" style={{ marginBottom: 16 }}>
            {createError}
          </div>
        )}

        <div className="card">
          <button
            className="btn-primary"
            type="button"
            style={{ width: '100%' }}
            disabled={
              pendingFiles.length === 0 ||
              createMutation.isPending ||
              (selectedOp.prompts ?? []).some((p) => !promptValues[p.key]?.trim())
            }
            onClick={() => createMutation.mutate()}
          >
            <Icon name="upload" size={14} />{' '}
            {createMutation.isPending
              ? 'Uploading…'
              : `Create Batch (${pendingFiles.length} file${pendingFiles.length === 1 ? '' : 's'})`}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Batch Processing</h1>
          <p>Process many files with one operation.</p>
        </div>
        <div className="page-header-actions">
          <button className="btn-secondary" type="button" onClick={startNewBatch}>
            <Icon name="upload" size={14} /> New Batch
          </button>
        </div>
      </div>

      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
          <div>
            <span className="card-title" style={{ marginRight: 8 }}>
              {opLabel}
            </span>
            <span
              style={{
                fontSize: 11,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                background: 'var(--surface-2)',
                padding: '2px 8px',
                borderRadius: 4,
              }}
            >
              {batch.id.slice(0, 8)}
            </span>
          </div>
          <StatusBadge status={batch.status as Status} />
        </div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 16 }}>
          Started {formatDate(batch.created_at)}
        </p>

        <div className="stat-mini-grid">
          <div className="stat-mini">
            <div className="stat-mini-value">{batch.total}</div>
            <div className="stat-mini-label">Total</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{batch.completed}</div>
            <div className="stat-mini-label">Completed</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{inProgress}</div>
            <div className="stat-mini-label">In Progress</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{batch.failed}</div>
            <div className="stat-mini-label">Failed</div>
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 6 }}>
          <span>{percent}% complete</span>
          <span style={{ color: 'var(--text-muted)' }}>
            {batch.completed} / {batch.total} files
          </span>
        </div>
        <div className="progress-bar" style={{ marginBottom: 16 }}>
          <div className="progress-bar-fill" style={{ width: `${percent}%` }} />
        </div>

        {batch.failed > 0 && (
          <div className="banner banner-warning">
            <span>{batch.failed} files failed. Review errors below or retry failed files.</span>
            <button
              className="btn-danger-ghost"
              type="button"
              disabled={retryMutation.isPending}
              onClick={() => retryMutation.mutate()}
            >
              {retryMutation.isPending ? 'Retrying…' : 'Retry all'}
            </button>
          </div>
        )}

        <div className="page-header-actions" style={{ marginBottom: 20 }}>
          <button
            className="btn-secondary"
            type="button"
            disabled={batch.failed === 0 || retryMutation.isPending}
            onClick={() => retryMutation.mutate()}
          >
            <Icon name="retry" size={14} /> Retry Failed
          </button>
          <button className="btn-primary" type="button" disabled={batch.completed === 0} onClick={downloadZip}>
            <Icon name="download" size={14} /> Download ZIP
          </button>
        </div>

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>File</th>
                <th>Status</th>
                <th>Attempts</th>
                <th>Output</th>
                <th>Error</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => {
                const inputFile = filesMap.get(item.input_file_id)
                const type = inputFile ? mimeTypeToLabel(inputFile.mime_type) : 'file'
                return (
                  <tr key={item.id}>
                    <td>
                      <div className="table-file-cell">
                        <FileTypeIcon type={type} />
                        <span className="table-file-name">{inputFile?.filename ?? item.input_file_id}</span>
                      </div>
                    </td>
                    <td>
                      <StatusBadge status={item.status as Status} />
                    </td>
                    <td>{item.attempts}</td>
                    <td>
                      {item.status === 'completed' && item.output_file_id ? (
                        <div className="table-actions">
                          <button type="button" onClick={() => downloadItemOutput(item)}>
                            Download
                          </button>
                        </div>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td style={{ color: item.last_error ? 'var(--error)' : undefined }}>{item.last_error ?? '—'}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          {itemsQuery.isLoading && <p className="empty-hint">Loading items…</p>}
          {!itemsQuery.isLoading && items.length === 0 && batch.failed === batch.total && (
            <p className="empty-hint">Every file was rejected before processing — check file types and try again.</p>
          )}
        </div>
      </div>
    </div>
  )
}
