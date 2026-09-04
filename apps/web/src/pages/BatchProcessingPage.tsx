import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { StatusBadge } from '../components/StatusBadge'
import { mockBatchItems } from '../lib/mockData'

export function BatchProcessingPage() {
  const total = mockBatchItems.length
  const completed = mockBatchItems.filter((i) => i.status === 'completed').length
  const processing = mockBatchItems.filter((i) => i.status === 'processing').length
  const failed = mockBatchItems.filter((i) => i.status === 'failed').length
  const queued = mockBatchItems.filter((i) => i.status === 'queued').length
  const percent = Math.round((completed / total) * 100)

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Batch Processing</h1>
          <p>Process hundreds of files with one operation.</p>
        </div>
        <div className="page-header-actions">
          <button className="btn-primary" type="button" disabled>
            <Icon name="upload" size={14} /> Create Batch
          </button>
        </div>
      </div>

      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
          <div>
            <span className="card-title" style={{ marginRight: 8 }}>
              Invoice Conversion
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
              BF-2026-00481
            </span>
          </div>
          <StatusBadge status="processing" />
        </div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 16 }}>
          DOCX + PDF → Searchable PDF · Started 8 minutes ago
        </p>

        <div className="stat-mini-grid">
          <div className="stat-mini">
            <div className="stat-mini-value">{total}</div>
            <div className="stat-mini-label">Total</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{completed}</div>
            <div className="stat-mini-label">Completed</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{processing}</div>
            <div className="stat-mini-label">Processing</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{failed}</div>
            <div className="stat-mini-label">Failed</div>
          </div>
          <div className="stat-mini">
            <div className="stat-mini-value">{queued}</div>
            <div className="stat-mini-label">Queued</div>
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 6 }}>
          <span>{percent}% complete</span>
          <span style={{ color: 'var(--text-muted)' }}>
            {completed} / {total} files
          </span>
        </div>
        <div className="progress-bar" style={{ marginBottom: 16 }}>
          <div className="progress-bar-fill" style={{ width: `${percent}%` }} />
        </div>

        {failed > 0 && (
          <div className="banner banner-warning">
            <span>
              {failed} files failed. Review errors below or retry failed files.
            </span>
            <button className="btn-danger-ghost" type="button" disabled>
              Retry all
            </button>
          </div>
        )}

        <div className="page-header-actions" style={{ marginBottom: 20 }}>
          <button className="btn-secondary" type="button" disabled>
            <Icon name="retry" size={14} /> Retry Failed
          </button>
          <button className="btn-secondary" type="button" disabled>
            <Icon name="download" size={14} /> Download Completed
          </button>
          <button className="btn-primary" type="button" disabled>
            <Icon name="download" size={14} /> Download ZIP
          </button>
        </div>

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>File</th>
                <th>Operation</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Output</th>
                <th>Error</th>
              </tr>
            </thead>
            <tbody>
              {mockBatchItems.map((item) => (
                <tr key={item.id}>
                  <td>
                    <div className="table-file-cell">
                      <FileTypeIcon type={item.type} />
                      <span className="table-file-name">{item.file}</span>
                    </div>
                  </td>
                  <td>{item.operation}</td>
                  <td>
                    <StatusBadge status={item.status} />
                  </td>
                  <td>{item.duration}</td>
                  <td>{item.output}</td>
                  <td style={{ color: item.error !== '—' ? 'var(--error)' : undefined }}>{item.error}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
