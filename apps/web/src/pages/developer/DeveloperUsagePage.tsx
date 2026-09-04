import { useState } from 'react'
import { Icon } from '../../components/Icon'

const ranges = ['24h', '7d', '30d', 'Custom']
const bars = [45, 70, 60, 55, 65, 72, 40, 50, 42, 38, 30, 48, 58, 45, 65, 40, 45, 55, 62, 58, 52, 68, 60, 55, 42, 48, 38, 45, 78, 72]

export function DeveloperUsagePage() {
  const [range, setRange] = useState('30d')

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

      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Total Requests</span>
            <span className="stat-tile-icon">
              <Icon name="api" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">84,291</div>
          <div className="stat-tile-caption">+12% vs prev period</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Successful</span>
            <span className="stat-tile-icon">
              <Icon name="check" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">83,620</div>
          <div className="stat-tile-caption">99.2% success rate</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Failed</span>
            <span className="stat-tile-icon">
              <Icon name="x" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">671</div>
          <div className="stat-tile-caption">0.8% error rate</div>
        </div>
      </div>

      <div className="card-row" style={{ marginBottom: 16 }}>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Files Processed</span>
            <span className="stat-tile-icon">
              <Icon name="usage" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">12,841</div>
          <div className="stat-tile-caption">via API</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Avg Processing</span>
            <span className="stat-tile-icon">
              <Icon name="usage" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">5.2s</div>
          <div className="stat-tile-caption">p95: 18.4s</div>
        </div>
        <div className="stat-tile">
          <div className="stat-tile-top">
            <span className="stat-tile-label">Data Processed</span>
            <span className="stat-tile-icon">
              <Icon name="usage" size={16} />
            </span>
          </div>
          <div className="stat-tile-value">48.2 GB</div>
          <div className="stat-tile-caption">input + output</div>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Requests over time · Last {range}</div>
        <div className="bar-chart">
          {bars.map((h, i) => (
            <div
              key={i}
              className={`bar-chart-bar ${i >= bars.length - 3 ? 'bar-chart-bar-active' : ''}`}
              style={{ height: `${h}%` }}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
