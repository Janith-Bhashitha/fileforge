import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { Icon } from './Icon'
import './Toast.css'

type ToastVariant = 'success' | 'error' | 'info'

interface Toast {
  id: number
  variant: ToastVariant
  message: string
}

interface ToastContextValue {
  show: (variant: ToastVariant, message: string) => void
  success: (message: string) => void
  error: (message: string) => void
  info: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined)

const ICONS: Record<ToastVariant, string> = { success: 'check', error: 'x', info: 'sparkles' }
const AUTO_DISMISS_MS = 4000

let nextId = 1

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const show = useCallback(
    (variant: ToastVariant, message: string) => {
      const id = nextId++
      setToasts((prev) => [...prev, { id, variant, message }])
      setTimeout(() => dismiss(id), AUTO_DISMISS_MS)
    },
    [dismiss]
  )

  const value: ToastContextValue = {
    show,
    success: (message) => show('success', message),
    error: (message) => show('error', message),
    info: (message) => show('info', message),
  }

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-stack" role="status" aria-live="polite">
        {toasts.map((t) => (
          <div className={`toast toast-${t.variant}`} key={t.id}>
            <Icon name={ICONS[t.variant] as Parameters<typeof Icon>[0]['name']} size={16} />
            <span>{t.message}</span>
            <button className="toast-dismiss" onClick={() => dismiss(t.id)} aria-label="Dismiss">
              <Icon name="x" size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) {
    throw new Error('useToast must be used within ToastProvider')
  }
  return ctx
}
