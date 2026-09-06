import { useQuery } from '@tanstack/react-query'
import { Icon } from '../components/Icon'
import { api } from '../lib/api'

interface HealthResponse {
  status: string
  db: string
}

export function SystemStatusPage() {
  const health = useQuery({
    queryKey: ['health'],
    queryFn: () => api.get<HealthResponse>('/healthz'),
    // The point of a status page is to be current, so it re-checks rather
    // than serving whatever the cache happened to hold.
    refetchInterval: 15000,
  })

  const apiOk = health.data?.status === 'ok'
  const dbOk = health.data?.db === 'ok'

  function label(ok: boolean) {
    if (health.isLoading) return 'Checking…'
    if (health.isError) return 'Unreachable'
    return ok ? 'Operational' : 'Degraded'
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>System Status</h1>
          <p>Live status of FileForge services.</p>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Services</div>
        <div className="service-row">
          <span className="service-name">
            <Icon name="api" size={16} />
            API
          </span>
          <span className={apiOk ? 'service-status-ok' : 'service-status-degraded'}>{label(apiOk)}</span>
        </div>
        <div className="service-row">
          <span className="service-name">
            <Icon name="database" size={16} />
            Database
          </span>
          <span className={dbOk ? 'service-status-ok' : 'service-status-degraded'}>{label(dbOk)}</span>
        </div>
      </div>
    </div>
  )
}
