import { useQuery } from '@tanstack/react-query'
import { Icon } from '../components/Icon'
import { api } from '../lib/api'

interface HealthResponse {
  status: string
  db: string
}

const previewServices = [
  { name: 'File Upload', phase: 'Phase 2' },
  { name: 'Conversion Workers', phase: 'Phase 3' },
  { name: 'AI Processing', phase: 'Phase 7' },
  { name: 'Webhooks', phase: 'Phase 9' },
]

export function SystemStatusPage() {
  const health = useQuery({
    queryKey: ['health'],
    queryFn: () => api.get<HealthResponse>('/healthz'),
    refetchInterval: 15000,
  })

  const apiOk = health.data?.status === 'ok'
  const dbOk = health.data?.db === 'ok'

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>System Status</h1>
          <p>Real-time status of FileForge services.</p>
        </div>
      </div>

      <div className="banner banner-info">
        This page previews the full monitoring dashboard planned for Phase 5. Only API and Database below reflect
        live status right now — the rest are shown as a preview of what's coming.
      </div>

      <div className="card">
        <div className="card-title">Live Services</div>
        <div className="service-row">
          <span className="service-name">
            <Icon name="api" size={16} />
            API
          </span>
          <span className={apiOk ? 'service-status-ok' : 'service-status-degraded'}>
            {health.isLoading ? 'Checking…' : apiOk ? 'Operational' : 'Unreachable'}
          </span>
        </div>
        <div className="service-row">
          <span className="service-name">
            <Icon name="database" size={16} />
            Database
          </span>
          <span className={dbOk ? 'service-status-ok' : 'service-status-degraded'}>
            {health.isLoading ? 'Checking…' : dbOk ? 'Operational' : 'Unreachable'}
          </span>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Planned Services (preview)</div>
        {previewServices.map((s) => (
          <div className="service-row" key={s.name}>
            <span className="service-name" style={{ opacity: 0.6 }}>
              <Icon name="status" size={16} />
              {s.name}
            </span>
            <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>Planned — {s.phase}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
