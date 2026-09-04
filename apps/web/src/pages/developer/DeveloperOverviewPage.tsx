import { Icon } from '../../components/Icon'

const bars = [40, 65, 55, 48, 58, 70, 35, 45, 40, 38, 60, 50, 55, 52, 45, 40, 30, 42, 38, 80, 75, 60]

export function DeveloperOverviewPage() {
  return (
    <div>
      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">API Requests</span>
            <span className="stat-tile-icon">
              <Icon name="api" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">84,291</div>
          <div className="stat-tile-caption">Last 30 days</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Files Processed</span>
            <span className="stat-tile-icon">
              <Icon name="check" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">12,841</div>
          <div className="stat-tile-caption">via API</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Avg Response</span>
            <span className="stat-tile-icon">
              <Icon name="usage" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">142ms</div>
          <div className="stat-tile-caption">p95: 890ms</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Error Rate</span>
            <span className="stat-tile-icon">
              <Icon name="x" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">0.8%</div>
          <div className="stat-tile-caption">Last 7 days</div>
        </div>
      </div>

      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="card">
          <div className="code-block-header">POST /v1/jobs</div>
          <div className="code-block">{`curl -X POST https://api.fileforge.dev/v1/jobs \\
  -H "Authorization: Bearer ff_live_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "file_id": "file_01J9A3kx",
    "operation": "pdf_to_images",
    "options": {
      "format": "jpg",
      "quality": 90
    }
  }'`}</div>
        </div>
        <div className="card">
          <div className="card-title">API Requests — Last 14 Days</div>
          <div className="bar-chart">
            {bars.slice(-14).map((h, i) => (
              <div
                key={i}
                className={`bar-chart-bar ${i === 13 ? 'bar-chart-bar-active' : ''}`}
                style={{ height: `${h}%` }}
              />
            ))}
          </div>
        </div>
      </div>

      <div className="page-header-actions">
        <button className="btn-primary" type="button" disabled>
          <Icon name="key" size={14} /> Create API Key
        </button>
        <button className="btn-secondary" type="button" disabled>
          View Documentation
        </button>
      </div>
    </div>
  )
}
