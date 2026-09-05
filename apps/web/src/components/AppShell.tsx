import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../lib/auth'
import { useDeveloperMode } from '../lib/developerMode'
import { api } from '../lib/api'
import { ThemeToggle } from './ThemeToggle'
import { Icon, type IconName } from './Icon'
import './AppShell.css'

interface MeResponse {
  id: string
  email: string
}

export interface ShellContext {
  me?: MeResponse
  meLoading: boolean
  meError: boolean
}

interface NavItem {
  path: string
  label: string
  icon: IconName
}

interface NavSection {
  label: string
  items: NavItem[]
}

const mainSections: NavSection[] = [
  {
    label: 'Main',
    items: [
      { path: '/', label: 'Dashboard', icon: 'dashboard' },
      { path: '/convert', label: 'Convert', icon: 'convert' },
      { path: '/batches', label: 'Batch Processing', icon: 'batch' },
      { path: '/files', label: 'Files', icon: 'files' },
      { path: '/history', label: 'History', icon: 'history' },
    ],
  },
  {
    label: 'Intelligent Processing',
    items: [
      { path: '/ai', label: 'AI Processing', icon: 'ai' },
      { path: '/ocr', label: 'OCR', icon: 'ocr' },
      { path: '/insights', label: 'Document Insights', icon: 'insights' },
    ],
  },
]

const developerSection: NavSection = {
  label: 'Developer',
  items: [
    { path: '/developer/overview', label: 'API', icon: 'api' },
    { path: '/developer/api-keys', label: 'API Keys', icon: 'key' },
    { path: '/developer/webhooks', label: 'Webhooks', icon: 'webhook' },
    { path: '/developer/usage', label: 'Usage', icon: 'usage' },
    { path: '/developer/cli', label: 'CLI', icon: 'cli' },
  ],
}

const systemSection: NavSection = {
  label: 'System',
  items: [
    { path: '/settings', label: 'Settings', icon: 'settings' },
    { path: '/help', label: 'Help', icon: 'help' },
    { path: '/status', label: 'System Status', icon: 'status' },
  ],
}

// All sections, including Developer, regardless of the toggle — used only
// to resolve a page title, since a direct link to /developer/* must still
// show a real title even if the sidebar itself is hiding that section.
const allSections: NavSection[] = [...mainSections, developerSection, systemSection]

function titleFor(pathname: string): string {
  for (const section of allSections) {
    for (const item of section.items) {
      if (item.path === pathname) return item.label
    }
  }
  return 'FileForge'
}

export function AppShell() {
  const location = useLocation()
  const { logout } = useAuth()
  const { developerMode } = useDeveloperMode()
  const me = useQuery({ queryKey: ['me'], queryFn: () => api.get<MeResponse>('/api/auth/me') })
  const title = titleFor(location.pathname)

  const visibleSections = developerMode ? [...mainSections, developerSection, systemSection] : [...mainSections, systemSection]

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <div className="sidebar-brand-mark">FF</div>
          <span>FileForge</span>
        </div>

        <nav className="sidebar-nav">
          {visibleSections.map((section) => (
            <div className="sidebar-section" key={section.label}>
              <span className="sidebar-nav-label">{section.label}</span>
              {section.items.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.path === '/'}
                  className={({ isActive }) => `sidebar-nav-item${isActive ? ' sidebar-nav-item-active' : ''}`}
                >
                  <Icon name={item.icon} size={18} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="sidebar-user">
            <div className="sidebar-avatar">{me.data?.email ? me.data.email[0].toUpperCase() : '?'}</div>
            <span className="sidebar-email">{me.data?.email}</span>
          </div>
          <button className="sidebar-logout" onClick={logout} aria-label="Log out" title="Log out">
            <Icon name="logout" size={16} />
          </button>
        </div>
      </aside>

      <div className="shell-main">
        <header className="topbar">
          <h2 className="topbar-title">{title}</h2>
          <ThemeToggle />
        </header>
        <main className="shell-content">
          <Outlet context={{ me: me.data, meLoading: me.isLoading, meError: me.isError } satisfies ShellContext} />
        </main>
      </div>
    </div>
  )
}
