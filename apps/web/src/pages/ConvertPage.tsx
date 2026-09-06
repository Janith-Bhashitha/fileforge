import { useRef, useState, type ChangeEvent } from 'react'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { api, ApiError } from '../lib/api'

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

// Some operations need a value only the user can supply (which pages, what
// watermark text, which password). `prompts` declares those as fields to
// render; their values are merged into `options` at submit time.
interface OptionPrompt {
  key: string
  label: string
  placeholder: string
  defaultValue?: string
  type?: 'text' | 'password'
}

interface OperationConfig {
  label: string
  operation: string
  options?: Record<string, string>
  accept: string
  implemented: boolean
  prompts?: OptionPrompt[]
  // Only pdf-merge needs more than one input file, so everything else stays
  // on the simpler single-file path rather than every operation paying for
  // multi-file bookkeeping it never uses.
  multiple?: boolean
}

interface OperationCategory {
  label: string
  operations: OperationConfig[]
}

// Grouped the way people actually think about these tools (convert into
// PDF, convert out of PDF, rearrange an existing PDF, lock it down, plain
// image tools) rather than one flat alphabetically-arbitrary list - the
// grouping iLovePDF and every other real tool in this space uses, because
// it's the grouping that maps to how someone searches for what they need.
const categories: OperationCategory[] = [
  {
    label: 'Convert to PDF',
    operations: [
      { label: 'JPG → PDF', operation: 'image-to-pdf', accept: 'image/jpeg', implemented: true },
      { label: 'PNG → PDF', operation: 'image-to-pdf', accept: 'image/png', implemented: true },
      { label: 'DOCX → PDF', operation: 'docx-to-pdf', accept: '.docx', implemented: true },
      { label: 'PPTX → PDF', operation: 'pptx-to-pdf', accept: '.pptx', implemented: true },
      { label: 'XLSX → PDF', operation: 'xlsx-to-pdf', accept: '.xlsx', implemented: true },
      { label: 'TXT → PDF', operation: 'txt-to-pdf', accept: '.txt', implemented: true },
    ],
  },
  {
    label: 'Convert from PDF',
    operations: [
      { label: 'PDF → JPG', operation: 'pdf-to-image', options: { format: 'jpeg' }, accept: 'application/pdf', implemented: true },
      { label: 'PDF → PNG', operation: 'pdf-to-image', options: { format: 'png' }, accept: 'application/pdf', implemented: true },
    ],
  },
  {
    label: 'Organize PDF',
    operations: [
      { label: 'Merge PDF', operation: 'pdf-merge', accept: 'application/pdf', implemented: true, multiple: true },
      { label: 'Split PDF', operation: 'pdf-split', accept: 'application/pdf', implemented: true },
      {
        label: 'Rotate PDF',
        operation: 'pdf-rotate',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'angle', label: 'Rotation', placeholder: '90', defaultValue: '90' }],
      },
      {
        label: 'Remove Pages',
        operation: 'pdf-remove-pages',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'pages', label: 'Pages to remove', placeholder: 'e.g. 1,3,5-7' }],
      },
      {
        label: 'Extract Pages',
        operation: 'pdf-extract-pages',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'pages', label: 'Pages to keep', placeholder: 'e.g. 1,3,5-7' }],
      },
    ],
  },
  {
    label: 'Optimize & Secure PDF',
    operations: [
      { label: 'Compress PDF', operation: 'pdf-compress', accept: 'application/pdf', implemented: true },
      {
        label: 'Watermark PDF',
        operation: 'pdf-watermark',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'text', label: 'Watermark text', placeholder: 'CONFIDENTIAL' }],
      },
      {
        label: 'Protect PDF',
        operation: 'pdf-protect',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'password', label: 'Password', placeholder: 'Choose a password', type: 'password' }],
      },
      {
        label: 'Unlock PDF',
        operation: 'pdf-unlock',
        accept: 'application/pdf',
        implemented: true,
        prompts: [{ key: 'password', label: 'Current password', placeholder: 'Enter the PDF password', type: 'password' }],
      },
    ],
  },
  {
    label: 'Image Tools',
    operations: [
      { label: 'JPG → PNG', operation: 'image-convert', options: { format: 'png' }, accept: 'image/jpeg', implemented: true },
      { label: 'PNG → JPG', operation: 'image-convert', options: { format: 'jpeg' }, accept: 'image/png', implemented: true },
      { label: 'Resize Image', operation: 'image-resize', options: { max_width: '1200' }, accept: 'image/*', implemented: true },
    ],
  },
]

const allOperations = categories.flatMap((c) => c.operations)

type Status = 'idle' | 'uploading' | 'converting' | 'done' | 'error'

function initialPromptValues(op: OperationConfig): Record<string, string> {
  const values: Record<string, string> = {}
  for (const p of op.prompts ?? []) values[p.key] = p.defaultValue ?? ''
  return values
}

export function ConvertPage() {
  const [selectedOp, setSelectedOp] = useState(allOperations[0])
  const [files, setFiles] = useState<File[]>([])
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')
  const [result, setResult] = useState<FileResponse | null>(null)
  const [promptValues, setPromptValues] = useState<Record<string, string>>({})
  const fileInputRef = useRef<HTMLInputElement>(null)

  function reset() {
    setFiles([])
    setStatus('idle')
    setError('')
    setResult(null)
  }

  function handleSelectOperation(op: OperationConfig) {
    setSelectedOp(op)
    setPromptValues(initialPromptValues(op))
    reset()
  }

  function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files ?? [])
    if (selected.length === 0) return
    setFiles((prev) => (selectedOp.multiple ? [...prev, ...selected] : [selected[0]]))
    setStatus('idle')
    setError('')
    setResult(null)
    e.target.value = ''
  }

  function removeFile(index: number) {
    setFiles((prev) => prev.filter((_, i) => i !== index))
  }

  async function handleConvert() {
    if (files.length === 0 || !selectedOp.implemented) return
    setStatus('uploading')
    setError('')

    try {
      const uploadedIds: string[] = []
      for (const file of files) {
        const formData = new FormData()
        formData.append('file', file)
        const uploaded = await api.upload<FileResponse>('/api/v1/files', formData)
        uploadedIds.push(uploaded.id)
      }

      setStatus('converting')
      const converted = await api.post<FileResponse>('/api/v1/convert', {
        // Multi-input operations (currently just pdf-merge) take file_ids in
        // the order they should be combined; everything else sends the one
        // file_id it's always sent.
        ...(uploadedIds.length > 1 ? { file_ids: uploadedIds } : { file_id: uploadedIds[0] }),
        operation: selectedOp.operation,
        version: 'v1',
        options: { ...(selectedOp.options ?? {}), ...promptValues },
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
  const missingPrompt = (selectedOp.prompts ?? []).some((p) => !promptValues[p.key]?.trim())
  const needsMoreFiles = selectedOp.multiple && files.length < 2

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
        {categories.map((category) => (
          <div key={category.label} style={{ marginBottom: 14 }}>
            <div
              style={{
                fontSize: 11,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--text-muted)',
                marginBottom: 8,
              }}
            >
              {category.label}
            </div>
            <div className="pill-group">
              {category.operations.map((op) => (
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
        ))}
      </div>

      {selectedOp.prompts && selectedOp.prompts.length > 0 && (
        <div className="card">
          {selectedOp.prompts.map((prompt) => (
            <div className="field-row" key={prompt.key}>
              <label htmlFor={`opt-${prompt.key}`}>{prompt.label}</label>
              <input
                id={`opt-${prompt.key}`}
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
        <div className="dropzone" onClick={() => fileInputRef.current?.click()} style={{ cursor: 'pointer' }}>
          <input
            ref={fileInputRef}
            type="file"
            multiple={selectedOp.multiple}
            accept={selectedOp.accept}
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">
            {files.length === 0
              ? selectedOp.multiple
                ? 'Click to choose files (in the order to merge them)'
                : 'Click to choose a file'
              : selectedOp.multiple
                ? `${files.length} file${files.length === 1 ? '' : 's'} selected — click to add more`
                : files[0].name}
          </div>
          {!selectedOp.multiple && files[0] && (
            <div className="dropzone-sub">{(files[0].size / 1024).toFixed(0)} KB</div>
          )}
          {!files.length && <div className="dropzone-sub">or browse from your device</div>}
        </div>

        {selectedOp.multiple && files.length > 0 && (
          <div style={{ marginTop: 16 }}>
            {files.map((f, i) => (
              <div
                key={`${f.name}-${i}`}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 0',
                  borderBottom: i < files.length - 1 ? '1px solid var(--border)' : undefined,
                }}
              >
                <div className="table-file-cell">
                  <span style={{ color: 'var(--text-muted)', fontSize: 12, width: 16 }}>{i + 1}</span>
                  <FileTypeIcon type="pdf" />
                  <span className="table-file-name">{f.name}</span>
                </div>
                <button className="btn-danger-ghost" type="button" onClick={() => removeFile(i)}>
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
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

      {needsMoreFiles && files.length === 1 && (
        <div className="banner banner-info">Add at least one more PDF to merge.</div>
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
            disabled={files.length === 0 || isBusy || !selectedOp.implemented || missingPrompt || needsMoreFiles}
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
