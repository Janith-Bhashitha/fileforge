import { useQuery } from '@tanstack/react-query'
import { Link, useOutletContext } from 'react-router-dom'
import { api } from '../lib/api'
import { Icon } from '../components/Icon'
import type { ShellContext } from '../components/AppShell'
import './DashboardPage.css'

interface HealthResponse {
  status: string
  db: string
}

function greeting(): string {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

// A sample of what the Convert page offers, not the full list - the point is
// to give someone landing here a way in, not to duplicate the whole menu.
const quickActions = [
  { from: 'JPG', to: 'PDF', label: 'Image → PDF' },
  { from: 'PDF', to: 'JPG', label: 'PDF → Image' },
  { from: 'DOCX', to: 'PDF', label: 'Word → PDF' },
  { from: 'PDF', to: 'PDF', label: 'Merge PDFs' },
]

export function DashboardPage() {
  const { me } = useOutletContext<ShellContext>()

  const health = useQuery({
    queryKey: ['health'],
    queryFn: () => api.get<HealthResponse>('/healthz'),
  })

  return (
    <div>
      <h1 className="dash-title">
        {greeting()}
        {me?.email ? `, ${me.email.split('@')[0]}` : ''}
      </h1>
      <p className="dash-subtitle">Convert, batch process and manage your files.</p>

      <div className="status-grid">
        <StatCard
          label="API"
          value={health.isLoading ? 'Checking…' : health.isError ? 'Unreachable' : (health.data?.status ?? '—')}
          ok={health.data?.status === 'ok'}
          icon={<Icon name="check" size={16} />}
        />
        <StatCard
          label="Database"
          value={health.isLoading ? 'Checking…' : health.isError ? 'Unreachable' : (health.data?.db ?? '—')}
          ok={health.data?.db === 'ok'}
          icon={<Icon name="database" size={16} />}
        />
        <StatCard
          label="Account"
          value={me ? me.id : '—'}
          ok={Boolean(me)}
          icon={<Icon name="user" size={16} />}
          mono
        />
      </div>

      <div className="upcoming-card">
        <div className="upcoming-header">
          <h2 className="upcoming-title">Quick actions</h2>
          <Link to="/convert" className="upcoming-badge">
            All tools
          </Link>
        </div>
        <div className="upcoming-grid">
          {quickActions.map((op) => (
            <Link className="upcoming-tile" key={op.label} to="/convert">
              <Icon name="convert" size={16} />
              <div>
                <span className="upcoming-tile-types">
                  {op.from} <span className="upcoming-arrow">→</span> {op.to}
                </span>
                <span className="upcoming-tile-label">{op.label}</span>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}

interface StatCardProps {
  label: string
  value: string
  ok: boolean
  icon: React.ReactNode
  mono?: boolean
}

function StatCard({ label, value, ok, icon, mono }: StatCardProps) {
  return (
    <div className="status-card">
      <div className="status-card-top">
        <span className="status-label">{label}</span>
        <span className={`status-icon ${ok ? 'status-icon-ok' : 'status-icon-pending'}`}>{icon}</span>
      </div>
      <span className={`status-value ${ok ? 'status-ok' : 'status-pending'} ${mono ? 'status-value-mono' : ''}`}>
        {value}
      </span>
    </div>
  )
}
