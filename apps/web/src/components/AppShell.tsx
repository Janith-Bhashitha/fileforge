import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../lib/auth'
import { useDeveloperMode } from '../lib/developerMode'
import { api, assetUrl } from '../lib/api'
import { ThemeToggle } from './ThemeToggle'
import { Icon, type IconName } from './Icon'
import './AppShell.css'

export interface MeResponse {
  id: string
  email: string
  display_name: string
  avatar_url?: string
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
      { path: '/dashboard', label: 'Dashboard', icon: 'dashboard' },
      { path: '/convert', label: 'Convert', icon: 'convert' },
      { path: '/edit-sign', label: 'Edit & Sign', icon: 'signature' },
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
    { path: '/developer/cli', label: 'Shell', icon: 'cli' },
  ],
}

const systemSection: NavSection = {
  label: 'System',
  items: [
    { path: '/settings', label: 'Settings', icon: 'settings' },
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

function initialFor(me?: MeResponse): string {
  if (me?.display_name) return me.display_name[0].toUpperCase()
  if (me?.email) return me.email[0].toUpperCase()
  return '?'
}

export function AppShell() {
  const location = useLocation()
  const { logout } = useAuth()
  const { developerMode } = useDeveloperMode()
  const me = useQuery({ queryKey: ['me'], queryFn: () => api.get<MeResponse>('/api/auth/me') })
  const title = titleFor(location.pathname)

  // The sidebar was previously just a permanently-shrunk icon rail below
  // 860px, with no way to ever see a label — not a mobile nav, just a
  // cramped desktop one. This makes it a real off-canvas drawer instead.
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  useEffect(() => {
    setMobileNavOpen(false)
  }, [location.pathname])

  const visibleSections = developerMode ? [...mainSections, developerSection, systemSection] : [...mainSections, systemSection]

  return (
    <div className="shell">
      {mobileNavOpen && <div className="sidebar-backdrop" onClick={() => setMobileNavOpen(false)} />}

      <aside className={`sidebar${mobileNavOpen ? ' sidebar-open' : ''}`}>
        <div className="sidebar-brand">
          <div className="sidebar-brand-mark">FF</div>
          <span>FileForge</span>
          <button
            className="sidebar-close"
            onClick={() => setMobileNavOpen(false)}
            aria-label="Close menu"
          >
            <Icon name="x" size={18} />
          </button>
        </div>

        <nav className="sidebar-nav">
          {visibleSections.map((section) => (
            <div className="sidebar-section" key={section.label}>
              <span className="sidebar-nav-label">{section.label}</span>
              {section.items.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.path === '/dashboard'}
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
            {me.data?.avatar_url ? (
              <img className="sidebar-avatar sidebar-avatar-img" src={assetUrl(me.data.avatar_url)} alt="" />
            ) : (
              <div className="sidebar-avatar">{initialFor(me.data)}</div>
            )}
            <div className="sidebar-user-text">
              <span className="sidebar-name">{me.data?.display_name || me.data?.email}</span>
              <span className="sidebar-email">{me.data?.email}</span>
            </div>
          </div>
          <button className="sidebar-logout" onClick={logout} aria-label="Log out" title="Log out">
            <Icon name="logout" size={16} />
          </button>
        </div>
      </aside>

      <div className="shell-main">
        <header className="topbar">
          <div className="topbar-left">
            <button
              className="hamburger-btn"
              onClick={() => setMobileNavOpen(true)}
              aria-label="Open menu"
            >
              <Icon name="menu" size={20} />
            </button>
            <h2 className="topbar-title">{title}</h2>
          </div>
          <ThemeToggle />
        </header>
        <main className="shell-content">
          <Outlet context={{ me: me.data, meLoading: me.isLoading, meError: me.isError } satisfies ShellContext} />
        </main>
      </div>
    </div>
  )
}
