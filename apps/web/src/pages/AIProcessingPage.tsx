import { useState } from 'react'
import { Icon } from '../components/Icon'

const exampleTasks = [
  'Convert all scanned invoices to searchable PDFs',
  'OCR these documents and extract key information',
  'Extract invoice numbers and dates from PDFs',
  'Rename files based on their content',
  'Classify these documents by type',
  'Compress all PDFs to under 1 MB',
]

export function AIProcessingPage() {
  const [instruction, setInstruction] = useState('')

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>AI Processing</h1>
          <p>Describe what you want to do with your files.</p>
        </div>
      </div>

      <div className="card">
        <div className="dropzone">
          <div className="dropzone-icon">
            <Icon name="upload" size={20} />
          </div>
          <div className="dropzone-title">Drop files for AI processing</div>
          <div className="dropzone-sub">or click to browse</div>
        </div>
      </div>

      <div className="card">
        <div className="field-row">
          <label htmlFor="instruction">What would you like to do?</label>
          <textarea
            id="instruction"
            rows={3}
            value={instruction}
            onChange={(e) => setInstruction(e.target.value)}
            placeholder="e.g. Convert all scanned invoices to searchable PDFs, extract invoice number and date, rename the files, and create a ZIP."
            style={{
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-sm)',
              padding: '12px',
              fontSize: 14,
              color: 'var(--text)',
              resize: 'vertical',
              fontFamily: 'inherit',
            }}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
            AI will generate a plan for your review before processing begins.
          </p>
          <button className="btn-primary" type="button" disabled>
            <Icon name="sparkles" size={14} /> Process with AI — Phase 8
          </button>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Example tasks</div>
        <div className="pill-group">
          {exampleTasks.map((task) => (
            <button
              key={task}
              type="button"
              className="pill-option"
              onClick={() => setInstruction(task)}
            >
              {task}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
