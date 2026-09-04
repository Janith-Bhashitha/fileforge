import { useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Icon } from '../components/Icon'
import type { ShellContext } from '../components/AppShell'

const navItems = [
  { icon: 'user', label: 'Profile', enabled: true },
  { icon: 'lock', label: 'Security', enabled: false },
  { icon: 'database', label: 'Storage', enabled: false },
  { icon: 'settings', label: 'Processing Defaults', enabled: false },
  { icon: 'api', label: 'Billing & Plan', enabled: false },
] as const

export function SettingsPage() {
  const { me } = useOutletContext<ShellContext>()
  const [name, setName] = useState(me?.email?.split('@')[0] ?? '')
  const [email, setEmail] = useState(me?.email ?? '')

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Settings</h1>
        </div>
      </div>

      <div className="settings-layout">
        <div className="settings-nav">
          {navItems.map((item) => (
            <div
              key={item.label}
              className={`settings-nav-item ${item.enabled ? 'settings-nav-item-active' : 'settings-nav-item-disabled'}`}
            >
              <Icon name={item.icon} size={16} />
              <span>{item.label}</span>
            </div>
          ))}
        </div>

        <div className="card">
          <div className="card-title">Profile</div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 24 }}>
            <div
              style={{
                width: 56,
                height: 56,
                borderRadius: '50%',
                background: 'var(--accent)',
                color: 'var(--accent-contrast)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 700,
                fontSize: 20,
              }}
            >
              {email ? email[0].toUpperCase() : '?'}
            </div>
            <button className="btn-secondary" type="button" disabled>
              Change avatar
            </button>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
            <div className="field-row" style={{ marginBottom: 0 }}>
              <label htmlFor="name">Full Name</label>
              <input id="name" type="text" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="field-row" style={{ marginBottom: 0 }}>
              <label htmlFor="settings-email">Email</label>
              <input id="settings-email" type="text" value={email} onChange={(e) => setEmail(e.target.value)} disabled />
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 20 }}>
            <div className="field-row" style={{ marginBottom: 0 }}>
              <label htmlFor="tz">Timezone</label>
              <select id="tz" defaultValue="UTC">
                <option>UTC</option>
                <option>Pacific Time (PT)</option>
                <option>Eastern Time (ET)</option>
              </select>
            </div>
            <div className="field-row" style={{ marginBottom: 0 }}>
              <label htmlFor="lang">Language</label>
              <select id="lang" defaultValue="English">
                <option>English</option>
              </select>
            </div>
          </div>

          <button className="btn-primary" type="button" disabled>
            Save Changes
          </button>
        </div>
      </div>
    </div>
  )
}
