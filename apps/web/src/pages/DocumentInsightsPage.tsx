import { Icon } from '../components/Icon'

export function DocumentInsightsPage() {
  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Document Insights</h1>
          <p>Intelligent document classification, entity extraction, and metadata.</p>
        </div>
      </div>

      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="card">
          <div className="dropzone">
            <div className="dropzone-icon">
              <Icon name="upload" size={20} />
            </div>
            <div className="dropzone-title">Drop document here</div>
            <div className="dropzone-sub">PDF, DOCX, images</div>
          </div>
        </div>
        <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <p className="empty-hint" style={{ padding: 0 }}>
            <Icon name="insights" size={28} />
            <br />
            Upload a document to get started
          </p>
        </div>
      </div>

      <button className="btn-primary" type="button" disabled>
        <Icon name="insights" size={14} /> Analyze Document — Phase 7
      </button>
    </div>
  )
}
