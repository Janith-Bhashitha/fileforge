import { NavLink, Outlet } from 'react-router-dom'

const tabs = [
  { path: '/developer/overview', label: 'Overview' },
  { path: '/developer/api-keys', label: 'API Keys' },
  { path: '/developer/webhooks', label: 'Webhooks' },
  { path: '/developer/usage', label: 'Usage' },
  { path: '/developer/cli', label: 'CLI' },
]

export function DeveloperPage() {
  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Developer Platform</h1>
          <p>Integrate FileForge into your applications.</p>
        </div>
      </div>

      <div className="tabs">
        {tabs.map((tab) => (
          <NavLink
            key={tab.path}
            to={tab.path}
            className={({ isActive }) => `tab${isActive ? ' tab-active' : ''}`}
          >
            {tab.label}
          </NavLink>
        ))}
      </div>

      <Outlet />
    </div>
  )
}
