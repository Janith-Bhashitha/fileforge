import { useState } from 'react'
import { Icon } from '../components/Icon'

export function OCRPage() {
  const [searchablePdf, setSearchablePdf] = useState(true)
  const [plainText, setPlainText] = useState(true)
  const [preserveLayout, setPreserveLayout] = useState(false)
  const [structuredData, setStructuredData] = useState(false)

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>OCR</h1>
          <p>Turn scanned documents into searchable text.</p>
        </div>
      </div>

      <div className="card-row">
        <div className="card">
          <div className="dropzone">
            <div className="dropzone-icon">
              <Icon name="upload" size={20} />
            </div>
            <div className="dropzone-title">Drop document here</div>
            <div className="dropzone-sub">PDF, JPG, PNG, TIFF</div>
          </div>
        </div>
        <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <p className="empty-hint" style={{ padding: 0 }}>
            <Icon name="ocr" size={28} />
            <br />
            Upload a document to begin
          </p>
        </div>
      </div>

      <div className="card" style={{ maxWidth: 380 }}>
        <div className="card-title">OCR Configuration</div>

        <div className="field-row">
          <label htmlFor="lang">Language</label>
          <select id="lang" defaultValue="English">
            <option>English</option>
            <option>Spanish</option>
            <option>French</option>
            <option>German</option>
          </select>
        </div>

        <div className="field-row">
          <label htmlFor="pages">Page Range</label>
          <input id="pages" type="text" defaultValue="All pages" />
        </div>

        <label className="checkbox-row">
          <input type="checkbox" checked={searchablePdf} onChange={(e) => setSearchablePdf(e.target.checked)} />
          Create searchable PDF
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={plainText} onChange={(e) => setPlainText(e.target.checked)} />
          Extract plain text
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={preserveLayout} onChange={(e) => setPreserveLayout(e.target.checked)} />
          Preserve layout
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={structuredData} onChange={(e) => setStructuredData(e.target.checked)} />
          Extract structured data
        </label>

        <button className="btn-primary" type="button" disabled style={{ width: '100%', marginTop: 8 }}>
          <Icon name="ocr" size={14} /> Run OCR — Phase 7
        </button>
      </div>
    </div>
  )
}
