import { createContext, useContext, useState, type ReactNode } from 'react'

interface DeveloperModeContextValue {
  developerMode: boolean
  toggleDeveloperMode: () => void
}

const DeveloperModeContext = createContext<DeveloperModeContextValue | undefined>(undefined)

function getInitial(): boolean {
  return localStorage.getItem('developerMode') === 'true'
}

export function DeveloperModeProvider({ children }: { children: ReactNode }) {
  const [developerMode, setDeveloperMode] = useState<boolean>(getInitial)

  function toggleDeveloperMode() {
    setDeveloperMode((prev) => {
      const next = !prev
      localStorage.setItem('developerMode', String(next))
      return next
    })
  }

  return (
    <DeveloperModeContext.Provider value={{ developerMode, toggleDeveloperMode }}>
      {children}
    </DeveloperModeContext.Provider>
  )
}

export function useDeveloperMode() {
  const ctx = useContext(DeveloperModeContext)
  if (!ctx) {
    throw new Error('useDeveloperMode must be used within DeveloperModeProvider')
  }
  return ctx
}
