import { useCallback, useEffect, useRef, useState, type ChangeEvent, type PointerEvent as ReactPointerEvent } from 'react'
import * as pdfjsLib from 'pdfjs-dist'
import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'
import { useToast } from '../components/Toast'
import './EditSignPage.css'

// pdf.js does its parsing off the main thread. Vite's ?url import hands us
// the hashed asset path for the worker bundle, which is what keeps this
// working in a production build as well as in dev.
pdfjsLib.GlobalWorkerOptions.workerSrc = workerSrc

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

// Mirrors pdfops.OverlayElement on the API. x/y are PDF points from the
// bottom-left of the page and address the element's own bottom-left corner,
// which is the one convention shared end to end - the canvas converts into
// it on placement and never deals in it again.
interface OverlayElement {
  id: string
  type: 'text' | 'image'
  page: number
  x: number
  y: number
  text?: string
  size?: number
  color?: string
  data?: string
  width?: number
}

type Tool = 'text' | 'signature'

// Rendering above 1:1 keeps the preview legible on a high-DPI screen; the
// scale is divided back out when converting a click into PDF points, so it
// never leaks into the coordinates sent to the API.
const RENDER_SCALE = 1.5

export function EditSignPage() {
  const toast = useToast()

  const [file, setFile] = useState<File | null>(null)
  const [pdfDoc, setPdfDoc] = useState<pdfjsLib.PDFDocumentProxy | null>(null)
  const [pageNumber, setPageNumber] = useState(1)
  const [pageSize, setPageSize] = useState({ width: 0, height: 0 })
  const [elements, setElements] = useState<OverlayElement[]>([])
  const [tool, setTool] = useState<Tool>('text')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<FileResponse | null>(null)

  const [textValue, setTextValue] = useState('')
  const [textSize, setTextSize] = useState(14)
  const [textColor, setTextColor] = useState('#000000')

  const [signature, setSignature] = useState<string | null>(null)
  const [signatureWidth, setSignatureWidth] = useState(160)

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  // Guards against two renders racing onto the same canvas: pdf.js rejects a
  // render that starts while another is still running on the same context.
  const renderTaskRef = useRef<pdfjsLib.RenderTask | null>(null)

  function resetDocument() {
    setPdfDoc(null)
    setPageNumber(1)
    setElements([])
    setResult(null)
    setPageSize({ width: 0, height: 0 })
  }

  async function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0]
    e.target.value = ''
    if (!selected) return

    resetDocument()
    setFile(selected)

    try {
      const bytes = await selected.arrayBuffer()
      const doc = await pdfjsLib.getDocument({ data: bytes }).promise
      setPdfDoc(doc)
    } catch {
      toast.error('Could not read that PDF')
      setFile(null)
    }
  }

  const renderPage = useCallback(async () => {
    const canvas = canvasRef.current
    if (!pdfDoc || !canvas) return

    renderTaskRef.current?.cancel()

    const page = await pdfDoc.getPage(pageNumber)
    const viewport = page.getViewport({ scale: RENDER_SCALE })
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    canvas.width = viewport.width
    canvas.height = viewport.height
    setPageSize({ width: viewport.width, height: viewport.height })

    const task = page.render({ canvas, canvasContext: ctx, viewport })
    renderTaskRef.current = task
    try {
      await task.promise
    } catch {
      // A cancelled render is the expected outcome of paging quickly.
    }
  }, [pdfDoc, pageNumber])

  useEffect(() => {
    renderPage()
    return () => renderTaskRef.current?.cancel()
  }, [renderPage])

  // A click places the active tool's element with its bottom-left corner at
  // the pointer. The canvas may be laid out smaller than its backing store,
  // so positions are taken as a fraction of the displayed box rather than
  // from raw client offsets.
  function handleCanvasClick(e: React.MouseEvent<HTMLCanvasElement>) {
    if (!pdfDoc) return

    const rect = e.currentTarget.getBoundingClientRect()
    const xCanvas = ((e.clientX - rect.left) / rect.width) * pageSize.width
    const yCanvas = ((e.clientY - rect.top) / rect.height) * pageSize.height

    const x = xCanvas / RENDER_SCALE
    // Canvas y grows downward, PDF y grows upward from the bottom.
    const y = (pageSize.height - yCanvas) / RENDER_SCALE

    if (tool === 'text') {
      if (!textValue.trim()) {
        toast.error('Type some text first')
        return
      }
      setElements((prev) => [
        ...prev,
        { id: crypto.randomUUID(), type: 'text', page: pageNumber, x, y, text: textValue, size: textSize, color: textColor },
      ])
      return
    }

    if (!signature) {
      toast.error('Draw a signature first')
      return
    }
    setElements((prev) => [
      ...prev,
      { id: crypto.randomUUID(), type: 'image', page: pageNumber, x, y, data: signature, width: signatureWidth },
    ])
  }

  async function handleApply() {
    if (!file || elements.length === 0) return
    setBusy(true)
    setResult(null)

    try {
      const formData = new FormData()
      formData.append('file', file)
      const uploaded = await api.upload<FileResponse>('/api/v1/files', formData)

      // `id` is local bookkeeping for the placed-elements list; the API
      // neither wants nor understands it.
      const payload = elements.map((el) => ({
        type: el.type,
        page: el.page,
        x: el.x,
        y: el.y,
        ...(el.type === 'text'
          ? { text: el.text, size: el.size, color: el.color }
          : { data: el.data, width: el.width }),
      }))
      const edited = await api.post<FileResponse>('/api/v1/convert', {
        file_id: uploaded.id,
        operation: 'pdf-overlay',
        version: 'v1',
        options: { elements: JSON.stringify(payload) },
      })

      setResult(edited)
      toast.success('PDF updated')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Could not apply your edits')
    } finally {
      setBusy(false)
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

  const pageElements = elements.filter((el) => el.page === pageNumber)

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Edit &amp; Sign</h1>
          <p>Add text, stamps and signatures directly onto any page.</p>
        </div>
        <div className="page-header-actions">
          <input
            ref={fileInputRef}
            type="file"
            accept="application/pdf"
            style={{ display: 'none' }}
            onChange={handleFileChange}
          />
          <button className="btn-secondary" type="button" onClick={() => fileInputRef.current?.click()}>
            <Icon name="upload" size={14} /> {file ? 'Choose another' : 'Open PDF'}
          </button>
        </div>
      </div>

      {!pdfDoc ? (
        <div className="card">
          <div className="dropzone" style={{ cursor: 'pointer' }} onClick={() => fileInputRef.current?.click()}>
            <div className="dropzone-icon">
              <Icon name="signature" size={20} />
            </div>
            <div className="dropzone-title">Open a PDF to start editing</div>
            <div className="dropzone-sub">
              Your edits are composited on top of the page — the original content is never altered.
            </div>
          </div>
        </div>
      ) : (
        <div className="editsign-layout">
          <div className="card editsign-tools">
            <div className="card-title">Tools</div>
            <div className="pill-group" style={{ marginBottom: 16 }}>
              <button
                type="button"
                className={`pill-option ${tool === 'text' ? 'pill-option-active' : ''}`}
                onClick={() => setTool('text')}
              >
                Text
              </button>
              <button
                type="button"
                className={`pill-option ${tool === 'signature' ? 'pill-option-active' : ''}`}
                onClick={() => setTool('signature')}
              >
                Signature
              </button>
            </div>

            {tool === 'text' ? (
              <>
                <div className="field-row">
                  <label htmlFor="overlay-text">Text</label>
                  <input
                    id="overlay-text"
                    type="text"
                    value={textValue}
                    placeholder="Type, then click the page"
                    onChange={(e) => setTextValue(e.target.value)}
                  />
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                  <div className="field-row">
                    <label htmlFor="overlay-size">Size</label>
                    <input
                      id="overlay-size"
                      type="number"
                      min={6}
                      max={96}
                      value={textSize}
                      onChange={(e) => setTextSize(Number(e.target.value) || 12)}
                    />
                  </div>
                  <div className="field-row">
                    <label htmlFor="overlay-color">Colour</label>
                    <input
                      id="overlay-color"
                      type="color"
                      value={textColor}
                      onChange={(e) => setTextColor(e.target.value)}
                    />
                  </div>
                </div>
              </>
            ) : (
              <SignaturePad
                signature={signature}
                width={signatureWidth}
                onWidthChange={setSignatureWidth}
                onChange={setSignature}
              />
            )}

            <p className="editsign-hint">
              Click anywhere on the page to place the {tool === 'text' ? 'text' : 'signature'}.
            </p>

            <div className="card-title" style={{ marginTop: 20 }}>
              Placed ({elements.length})
            </div>
            {elements.length === 0 ? (
              <p className="editsign-hint">Nothing placed yet.</p>
            ) : (
              <ul className="editsign-list">
                {elements.map((el) => (
                  <li key={el.id}>
                    <span>
                      {el.type === 'text' ? `“${el.text}”` : 'Signature'} · p{el.page}
                    </span>
                    <button
                      type="button"
                      className="btn-icon"
                      title="Remove"
                      onClick={() => setElements((prev) => prev.filter((x) => x.id !== el.id))}
                    >
                      <Icon name="trash" size={13} />
                    </button>
                  </li>
                ))}
              </ul>
            )}

            <button
              className="btn-primary"
              type="button"
              style={{ marginTop: 16, width: '100%' }}
              disabled={elements.length === 0 || busy}
              onClick={handleApply}
            >
              {busy ? 'Applying…' : 'Apply & Save'}
            </button>

            {result && (
              <button className="btn-secondary" type="button" style={{ marginTop: 10, width: '100%' }} onClick={handleDownload}>
                <Icon name="download" size={14} /> Download result
              </button>
            )}
          </div>

          <div className="card editsign-stage">
            <div className="editsign-pager">
              <button
                type="button"
                className="btn-icon"
                disabled={pageNumber <= 1}
                onClick={() => setPageNumber((p) => Math.max(1, p - 1))}
              >
                <Icon name="chevron" size={14} />
              </button>
              <span>
                Page {pageNumber} of {pdfDoc.numPages}
              </span>
              <button
                type="button"
                className="btn-icon"
                disabled={pageNumber >= pdfDoc.numPages}
                onClick={() => setPageNumber((p) => Math.min(pdfDoc.numPages, p + 1))}
              >
                <Icon name="chevron" size={14} />
              </button>
            </div>

            <div className="editsign-canvas-wrap">
              <canvas ref={canvasRef} className="editsign-canvas" onClick={handleCanvasClick} />
              {/* Previews sit in the same bottom-left space the API uses, so
                  what is shown here is where the element actually lands. */}
              {pageElements.map((el) => (
                <div
                  key={el.id}
                  className="editsign-placed"
                  style={{
                    left: `${(el.x * RENDER_SCALE * 100) / (pageSize.width || 1)}%`,
                    bottom: `${(el.y * RENDER_SCALE * 100) / (pageSize.height || 1)}%`,
                  }}
                >
                  {el.type === 'text' ? (
                    <span style={{ fontSize: (el.size ?? 12) * RENDER_SCALE, color: el.color, lineHeight: 1 }}>
                      {el.text}
                    </span>
                  ) : (
                    <img src={el.data} alt="" style={{ width: (el.width ?? 160) * RENDER_SCALE }} />
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

interface SignaturePadProps {
  signature: string | null
  width: number
  onWidthChange: (width: number) => void
  onChange: (dataUrl: string | null) => void
}

// A plain pointer-driven scribble pad. The stroke is captured at the canvas's
// own resolution and exported as a PNG data URI, which is exactly what the
// API's image element takes - no upload round-trip for something this small.
function SignaturePad({ signature, width, onWidthChange, onChange }: SignaturePadProps) {
  const padRef = useRef<HTMLCanvasElement>(null)
  const drawing = useRef(false)

  function positionOf(e: ReactPointerEvent<HTMLCanvasElement>) {
    const canvas = e.currentTarget
    const rect = canvas.getBoundingClientRect()
    return {
      x: ((e.clientX - rect.left) / rect.width) * canvas.width,
      y: ((e.clientY - rect.top) / rect.height) * canvas.height,
    }
  }

  function start(e: ReactPointerEvent<HTMLCanvasElement>) {
    const ctx = padRef.current?.getContext('2d')
    if (!ctx) return
    drawing.current = true
    e.currentTarget.setPointerCapture(e.pointerId)
    const { x, y } = positionOf(e)
    ctx.beginPath()
    ctx.moveTo(x, y)
  }

  function move(e: ReactPointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return
    const ctx = padRef.current?.getContext('2d')
    if (!ctx) return
    const { x, y } = positionOf(e)
    ctx.lineTo(x, y)
    ctx.strokeStyle = '#111827'
    ctx.lineWidth = 2.5
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
    ctx.stroke()
  }

  function end() {
    if (!drawing.current) return
    drawing.current = false
    const canvas = padRef.current
    if (canvas) onChange(canvas.toDataURL('image/png'))
  }

  function clear() {
    const canvas = padRef.current
    const ctx = canvas?.getContext('2d')
    if (!canvas || !ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    onChange(null)
  }

  return (
    <>
      <div className="field-row">
        <label>Draw your signature</label>
        <canvas
          ref={padRef}
          width={360}
          height={140}
          className="editsign-pad"
          onPointerDown={start}
          onPointerMove={move}
          onPointerUp={end}
          onPointerLeave={end}
        />
      </div>
      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end' }}>
        <div className="field-row" style={{ flex: 1, marginBottom: 0 }}>
          <label htmlFor="sig-width">Width on page (pt)</label>
          <input
            id="sig-width"
            type="number"
            min={40}
            max={400}
            value={width}
            onChange={(e) => onWidthChange(Number(e.target.value) || 160)}
          />
        </div>
        <button className="btn-secondary" type="button" onClick={clear}>
          Clear
        </button>
      </div>
      {!signature && <p className="editsign-hint">Draw above, then click the page to place it.</p>}
    </>
  )
}
