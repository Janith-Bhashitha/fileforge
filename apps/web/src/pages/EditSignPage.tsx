import { useCallback, useEffect, useRef, useState, type ChangeEvent, type PointerEvent as ReactPointerEvent } from 'react'
import * as pdfjsLib from 'pdfjs-dist'
import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { Icon } from '../components/Icon'
import { api, ApiError } from '../lib/api'
import { useToast } from '../components/Toast'
import './EditSignPage.css'

// Vite's ?url import gives the hashed asset path for the worker bundle,
// which is what makes this work in a production build as well as dev.
pdfjsLib.GlobalWorkerOptions.workerSrc = workerSrc

interface FileResponse {
  id: string
  filename: string
  mime_type: string
  size: number
}

// Mirrors pdfops.OverlayElement. x/y are PDF points from the bottom-left of
// the page, addressing the element's own bottom-left corner.
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

// Above 1:1 for legibility on high-DPI screens. Divided back out when a
// click becomes PDF points, so it never reaches the API.
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
  const [signatureMode, setSignatureMode] = useState<'draw' | 'upload'>('draw')

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  // pdf.js rejects a render that starts while another is running on the
  // same context.
  const renderTaskRef = useRef<pdfjsLib.RenderTask | null>(null)

  const [draggingId, setDraggingId] = useState<string | null>(null)
  // A ref, not state: pointermove fires on every pixel of movement, and this
  // only needs to survive between events, not trigger a render itself.
  const dragRef = useRef<{ id: string; startPointerX: number; startPointerY: number; startElX: number; startElY: number } | null>(
    null
  )

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

  // The canvas may be laid out smaller than its backing store, so a client
  // coordinate is taken as a fraction of the displayed box, not a raw
  // offset. Shared by placement and dragging so both agree on where the
  // pointer actually is in PDF points.
  function pdfPointFromClient(clientX: number, clientY: number) {
    const canvas = canvasRef.current
    if (!canvas) return { x: 0, y: 0 }
    const rect = canvas.getBoundingClientRect()
    const xCanvas = ((clientX - rect.left) / rect.width) * pageSize.width
    const yCanvas = ((clientY - rect.top) / rect.height) * pageSize.height
    return {
      x: xCanvas / RENDER_SCALE,
      // Canvas y grows downward, PDF y grows upward from the bottom.
      y: (pageSize.height - yCanvas) / RENDER_SCALE,
    }
  }

  function handleCanvasClick(e: React.MouseEvent<HTMLCanvasElement>) {
    if (!pdfDoc) return

    const { x, y } = pdfPointFromClient(e.clientX, e.clientY)

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

  // Drag-to-move for an already-placed element. Pointer capture keeps the
  // move/up events targeted at this element even once the pointer leaves
  // its (small) bounding box, which a fast drag does constantly.
  function handleElementPointerDown(e: ReactPointerEvent<HTMLDivElement>, el: OverlayElement) {
    e.stopPropagation()
    e.currentTarget.setPointerCapture(e.pointerId)
    const start = pdfPointFromClient(e.clientX, e.clientY)
    dragRef.current = { id: el.id, startPointerX: start.x, startPointerY: start.y, startElX: el.x, startElY: el.y }
    setDraggingId(el.id)
  }

  function handleElementPointerMove(e: ReactPointerEvent<HTMLDivElement>) {
    const drag = dragRef.current
    if (!drag) return
    const current = pdfPointFromClient(e.clientX, e.clientY)
    const x = drag.startElX + (current.x - drag.startPointerX)
    const y = drag.startElY + (current.y - drag.startPointerY)
    setElements((prev) => prev.map((item) => (item.id === drag.id ? { ...item, x, y } : item)))
  }

  function handleElementPointerUp(e: ReactPointerEvent<HTMLDivElement>) {
    if (e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId)
    }
    dragRef.current = null
    setDraggingId(null)
  }

  async function handleApply() {
    if (!file || elements.length === 0) return
    setBusy(true)
    setResult(null)

    try {
      const formData = new FormData()
      formData.append('file', file)
      const uploaded = await api.upload<FileResponse>('/api/v1/files', formData)

      // `id` is local bookkeeping for the list; the API has no use for it.
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
              <>
                <div className="pill-group" style={{ marginBottom: 12 }}>
                  <button
                    type="button"
                    className={`pill-option ${signatureMode === 'draw' ? 'pill-option-active' : ''}`}
                    onClick={() => setSignatureMode('draw')}
                  >
                    Draw
                  </button>
                  <button
                    type="button"
                    className={`pill-option ${signatureMode === 'upload' ? 'pill-option-active' : ''}`}
                    onClick={() => setSignatureMode('upload')}
                  >
                    Upload image
                  </button>
                </div>

                {signatureMode === 'draw' ? (
                  <SignaturePad onChange={setSignature} />
                ) : (
                  <SignatureUpload signature={signature} onChange={setSignature} />
                )}

                <div className="field-row" style={{ marginTop: 12 }}>
                  <label htmlFor="sig-width">Width on page (pt)</label>
                  <input
                    id="sig-width"
                    type="number"
                    min={40}
                    max={400}
                    value={signatureWidth}
                    onChange={(e) => setSignatureWidth(Number(e.target.value) || 160)}
                  />
                </div>

                {!signature && (
                  <p className="editsign-hint">
                    {signatureMode === 'draw' ? 'Draw a signature above' : 'Upload an image above'}, then click the
                    page to place it.
                  </p>
                )}
              </>
            )}

            <p className="editsign-hint">
              Click anywhere on the page to place the {tool === 'text' ? 'text' : 'signature'} — placed items can be
              dragged to reposition them.
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
              {/* The only chevron points down, so each is rotated to face
                  the way it moves. */}
              <button
                type="button"
                className="btn-icon"
                aria-label="Previous page"
                title="Previous page"
                disabled={pageNumber <= 1}
                onClick={() => setPageNumber((p) => Math.max(1, p - 1))}
              >
                <span style={{ display: 'flex', transform: 'rotate(90deg)' }}>
                  <Icon name="chevron" size={14} />
                </span>
              </button>
              <span>
                Page {pageNumber} of {pdfDoc.numPages}
              </span>
              <button
                type="button"
                className="btn-icon"
                aria-label="Next page"
                title="Next page"
                disabled={pageNumber >= pdfDoc.numPages}
                onClick={() => setPageNumber((p) => Math.min(pdfDoc.numPages, p + 1))}
              >
                <span style={{ display: 'flex', transform: 'rotate(-90deg)' }}>
                  <Icon name="chevron" size={14} />
                </span>
              </button>
            </div>

            <div className="editsign-canvas-wrap">
              <canvas ref={canvasRef} className="editsign-canvas" onClick={handleCanvasClick} />
              {/* Previews use the same bottom-left space as the API, so this
                  is where the element actually lands. Draggable: pointer
                  events are enabled here (the CSS default for the class is
                  otherwise none, so a click that misses every element still
                  reaches the canvas and places a new one). */}
              {pageElements.map((el) => (
                <div
                  key={el.id}
                  className={`editsign-placed ${draggingId === el.id ? 'editsign-placed-dragging' : ''}`}
                  style={{
                    left: `${(el.x * RENDER_SCALE * 100) / (pageSize.width || 1)}%`,
                    bottom: `${(el.y * RENDER_SCALE * 100) / (pageSize.height || 1)}%`,
                  }}
                  onPointerDown={(e) => handleElementPointerDown(e, el)}
                  onPointerMove={handleElementPointerMove}
                  onPointerUp={handleElementPointerUp}
                  onPointerCancel={handleElementPointerUp}
                  title="Drag to move"
                >
                  {el.type === 'text' ? (
                    <span style={{ fontSize: (el.size ?? 12) * RENDER_SCALE, color: el.color, lineHeight: 1 }}>
                      {el.text}
                    </span>
                  ) : (
                    <img src={el.data} alt="" draggable={false} style={{ width: (el.width ?? 160) * RENDER_SCALE }} />
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
  onChange: (dataUrl: string | null) => void
}

// A plain pointer-driven scribble pad. The stroke is captured at the canvas's
// own resolution and exported as a PNG data URI, which is exactly what the
// API's image element takes - no upload round-trip for something this small.
function SignaturePad({ onChange }: SignaturePadProps) {
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
    <div className="field-row">
      {/* htmlFor can't target a <canvas>, so the name lives on the canvas. */}
      <span className="field-row-caption">Draw your signature</span>
      <canvas
        ref={padRef}
        width={360}
        height={140}
        className="editsign-pad"
        role="img"
        aria-label="Signature drawing area"
        onPointerDown={start}
        onPointerMove={move}
        onPointerUp={end}
        onPointerLeave={end}
      />
      <button className="btn-secondary" type="button" style={{ alignSelf: 'flex-start' }} onClick={clear}>
        Clear
      </button>
    </div>
  )
}

interface SignatureUploadProps {
  signature: string | null
  onChange: (dataUrl: string | null) => void
}

// Turns an uploaded signature photo into an overlay-ready PNG. Almost every
// real-world signature photo is ink on white or off-white paper; without
// this a JPEG upload would stamp an opaque white rectangle onto the page
// instead of just the signature.
function SignatureUpload({ signature, onChange }: SignatureUploadProps) {
  const [removeBackground, setRemoveBackground] = useState(true)
  const rawImageRef = useRef<HTMLImageElement | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const process = useCallback(
    (image: HTMLImageElement, matte: boolean) => {
      // Capped well under the API's 2 MiB decoded-image limit - a phone
      // photo at full resolution has no business being that large anyway.
      const maxDim = 800
      const scale = Math.min(1, maxDim / Math.max(image.naturalWidth, image.naturalHeight))
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(image.naturalWidth * scale)
      canvas.height = Math.round(image.naturalHeight * scale)
      const ctx = canvas.getContext('2d')
      if (!ctx) return
      ctx.drawImage(image, 0, 0, canvas.width, canvas.height)

      if (matte) {
        // A flat brightness threshold, not real matting - good enough for
        // ink on paper, not meant for a busy or textured background.
        const frame = ctx.getImageData(0, 0, canvas.width, canvas.height)
        const data = frame.data
        for (let i = 0; i < data.length; i += 4) {
          const brightness = (data[i] + data[i + 1] + data[i + 2]) / 3
          if (brightness > 235) data[i + 3] = 0
        }
        ctx.putImageData(frame, 0, 0)
      }

      onChange(canvas.toDataURL('image/png'))
    },
    [onChange]
  )

  function handleFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    const img = new Image()
    const objectUrl = URL.createObjectURL(file)
    img.onload = () => {
      rawImageRef.current = img
      process(img, removeBackground)
      URL.revokeObjectURL(objectUrl)
    }
    img.src = objectUrl
  }

  function toggleRemoveBackground(next: boolean) {
    setRemoveBackground(next)
    if (rawImageRef.current) process(rawImageRef.current, next)
  }

  function clear() {
    rawImageRef.current = null
    onChange(null)
  }

  return (
    <div className="field-row">
      <span className="field-row-caption">Upload a signature image</span>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg"
        style={{ display: 'none' }}
        onChange={handleFile}
      />

      {signature ? (
        <div className="editsign-upload-preview">
          <img src={signature} alt="Uploaded signature" />
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn-secondary" type="button" onClick={() => fileInputRef.current?.click()}>
              Replace
            </button>
            <button className="btn-secondary" type="button" onClick={clear}>
              Clear
            </button>
          </div>
        </div>
      ) : (
        <button className="btn-secondary" type="button" onClick={() => fileInputRef.current?.click()}>
          <Icon name="upload" size={14} /> Choose image
        </button>
      )}

      <label className="checkbox-row" style={{ marginTop: 4 }}>
        <input type="checkbox" checked={removeBackground} onChange={(e) => toggleRemoveBackground(e.target.checked)} />
        Remove white background
      </label>
    </div>
  )
}
