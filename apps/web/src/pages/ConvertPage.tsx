import { useState } from 'react'
import { Icon } from '../components/Icon'

const operations = [
  'JPG → PDF',
  'PNG → PDF',
  'PDF → JPG',
  'PDF → PNG',
  'DOCX → PDF',
  'PPTX → PDF',
  'XLSX → PDF',
  'TXT → PDF',
]

export function ConvertPage() {
  const [selected, setSelected] = useState(operations[0])

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
              key={op}
              type="button"
              className={`pill-option ${selected === op ? 'pill-option-active' : ''}`}
              onClick={() => setSelected(op)}
            >
              {op}
            </button>
          ))}
        </div>
      </div>

      <div className="card">
        <div className="dropzone">
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">Drop files here</div>
          <div className="dropzone-sub">or browse from your device</div>
          <p className="dropzone-sub" style={{ marginTop: 12 }}>
            Supported: PDF, DOCX, XLSX, PPTX, JPG, PNG, TXT
          </p>
        </div>
      </div>

      <div className="card">
        <button className="btn-primary" type="button" disabled style={{ width: '100%' }}>
          Start Conversion — available in Phase 2
        </button>
      </div>
    </div>
  )
}
