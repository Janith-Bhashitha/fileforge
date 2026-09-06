import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Icon } from '../../components/Icon'
import { api } from '../../lib/api'
import { formatBytes } from '../../lib/format'

interface UsageSummary {
  conversions: number
  uploads: number
  bytes_stored: number
}

export function DeveloperOverviewPage() {
  const usage = useQuery({
    queryKey: ['usage', '30d'],
    queryFn: () => api.get<UsageSummary>('/api/v1/usage?range=30d'),
  })

  const show = (value: string) => (usage.isLoading ? '—' : value)

  return (
    <div>
      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Conversions</span>
            <span className="stat-tile-icon">
              <Icon name="convert" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">{show((usage.data?.conversions ?? 0).toLocaleString())}</div>
          <div className="stat-tile-caption">Last 30 days</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Files Uploaded</span>
            <span className="stat-tile-icon">
              <Icon name="upload" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">{show((usage.data?.uploads ?? 0).toLocaleString())}</div>
          <div className="stat-tile-caption">Last 30 days</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Stored Now</span>
            <span className="stat-tile-icon">
              <Icon name="database" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">{show(formatBytes(usage.data?.bytes_stored ?? 0))}</div>
          <div className="stat-tile-caption">All files you hold</div>
        </div>
      </div>

      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="card">
          <div className="code-block-header">POST /api/v1/jobs</div>
          <div className="code-block">{`curl -X POST https://YOUR-HOST/api/v1/jobs \\
  -H "X-API-Key: ffk_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "file_id": "8f14e45f-ceea-4e0a-9d2b-1a3b5c7d9e01",
    "operation": "pdf-to-image",
    "version": "v1",
    "options": {
      "format": "jpeg"
    }
  }'`}</div>
        </div>
        <div className="card">
          <div className="card-title">Authentication</div>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.6 }}>
            Send your key as an <code>X-API-Key</code> header on any <code>/api/v1</code> request. Keys are shown
            once when created and stored only as a hash, so a lost key must be revoked and replaced.
          </p>
          <div className="pill-group" style={{ marginTop: 14 }}>
            <Link className="btn-secondary" to="/developer/api-keys" style={{ textDecoration: 'none' }}>
              <Icon name="key" size={14} /> Manage API keys
            </Link>
            <Link className="btn-secondary" to="/developer/usage" style={{ textDecoration: 'none' }}>
              <Icon name="usage" size={14} /> View usage
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
