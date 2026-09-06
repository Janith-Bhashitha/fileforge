import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Icon } from '../../components/Icon'
import { api } from '../../lib/api'
import { formatBytes } from '../../lib/format'

interface DailyPoint {
  date: string
  count: number
}

interface OperationCount {
  operation: string
  count: number
}

interface UsageSummary {
  range: string
  conversions: number
  uploads: number
  bytes_in: number
  bytes_stored: number
  daily: DailyPoint[]
  top_operations: OperationCount[]
}

const ranges = ['24h', '7d', '30d']

export function DeveloperUsagePage() {
  const [range, setRange] = useState('30d')

  const usage = useQuery({
    queryKey: ['usage', range],
    queryFn: () => api.get<UsageSummary>(`/api/v1/usage?range=${range}`),
  })

  const data = usage.data
  const busiest = Math.max(1, ...(data?.daily ?? []).map((d) => d.count))

  return (
    <div>
      <div className="pill-group" style={{ marginBottom: 20 }}>
        {ranges.map((r) => (
          <button
            key={r}
            type="button"
            className={`pill-option ${range === r ? 'pill-option-active' : ''}`}
            onClick={() => setRange(r)}
          >
            {r}
          </button>
        ))}
      </div>

      {usage.isError && <p className="empty-hint">Could not load usage.</p>}

      <div className="card-row" style={{ marginBottom: 16 }}>
        <Tile
          label="Conversions"
          icon="convert"
          value={usage.isLoading ? '—' : (data?.conversions ?? 0).toLocaleString()}
          caption={`Last ${range}`}
        />
        <Tile
          label="Files Uploaded"
          icon="upload"
          value={usage.isLoading ? '—' : (data?.uploads ?? 0).toLocaleString()}
          caption={`Last ${range}`}
        />
        <Tile
          label="Data Uploaded"
          icon="usage"
          value={usage.isLoading ? '—' : formatBytes(data?.bytes_in ?? 0)}
          caption={`Last ${range}`}
        />
        <Tile
          label="Stored Now"
          icon="database"
          value={usage.isLoading ? '—' : formatBytes(data?.bytes_stored ?? 0)}
          caption="All files you hold"
        />
      </div>

      <div className="card">
        <div className="card-title">Conversions per day · Last {range}</div>
        {!usage.isLoading && (data?.daily.length ?? 0) === 0 ? (
          <p className="empty-hint">Nothing converted in this period yet.</p>
        ) : (
          <div className="bar-chart">
            {(data?.daily ?? []).map((d) => (
              <div
                key={d.date}
                className="bar-chart-bar bar-chart-bar-active"
                style={{ height: `${Math.round((d.count / busiest) * 100)}%` }}
                title={`${d.date}: ${d.count}`}
              />
            ))}
          </div>
        )}
      </div>

      <div className="card">
        <div className="card-title">Most used operations · Last {range}</div>
        {!usage.isLoading && (data?.top_operations.length ?? 0) === 0 ? (
          <p className="empty-hint">No operations run in this period yet.</p>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Operation</th>
                  <th>Runs</th>
                </tr>
              </thead>
              <tbody>
                {(data?.top_operations ?? []).map((o) => (
                  <tr key={o.operation}>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{o.operation}</td>
                    <td>{o.count.toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

interface TileProps {
  label: string
  value: string
  caption: string
  icon: 'convert' | 'upload' | 'usage' | 'database'
}

function Tile({ label, value, caption, icon }: TileProps) {
  return (
    <div className="stat-tile">
      <div className="stat-tile-top">
        <span className="stat-tile-label">{label}</span>
        <span className="stat-tile-icon">
          <Icon name={icon} size={16} />
        </span>
      </div>
      <div className="stat-tile-value">{value}</div>
      <div className="stat-tile-caption">{caption}</div>
    </div>
  )
}
